package send

import (
	"errors"
	"fmt"
	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/dto/parsemail"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/array"
	"github.com/Jinnrry/pmail/utils/async"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/maildomain"
	"github.com/Jinnrry/pmail/utils/smtp"
	log "github.com/sirupsen/logrus"
	"net"
	"net/textproto"
	"sync"
)

type mxDomain struct {
	domain string
	mxHost string
}

// mxFailoverEntry 记录一个收件人域名的全部 MX 主机（按优先级排序）。
type mxFailoverEntry struct {
	domain  string
	mxHosts []string
}

type temporaryMXFallbackError struct {
	lookupErr   error
	fallbackErr error
}

func (e *temporaryMXFallbackError) Error() string {
	return fmt.Sprintf("temporary MX lookup failed (%v); fallback delivery failed: %v", e.lookupErr, e.fallbackErr)
}

func (e *temporaryMXFallbackError) Unwrap() error {
	return e.lookupErr
}

// Forward 转发邮件
func Forward(ctx *context.Context, e *parsemail.Email, forwardAddress string, user *models.User) error {

	log.WithContext(ctx).Debugf("开始转发邮件")

	b := e.ForwardBuildBytes(ctx, user, forwardAddress)

	log.WithContext(ctx).Debugf("Forward message email_id=%d bytes=%d", e.MessageId, len(b))

	from := user.Account + "@" + config.Instance.Domains[0]
	return forwardData(ctx, config.Instance.Domains[0], b, forwardAddress, from)
}

func ForwardRaw(ctx *context.Context, e *parsemail.Email, rawEmailData []byte, forwardAddress string, user *models.User) error {
	log.WithContext(ctx).Debugf("开始原始邮件转发")

	from := user.Account + "@" + config.Instance.Domains[0]
	return forwardData(ctx, config.Instance.Domains[0], rawEmailData, forwardAddress, from)
}

func forwardData(ctx *context.Context, fromDomain string, data []byte, forwardAddress string, from string) error {
	var to []*parsemail.User
	to = []*parsemail.User{
		{EmailAddress: forwardAddress},
	}

	err, _ := doSend(ctx, fromDomain, data, to, from)
	return err
}

func Send(ctx *context.Context, e *parsemail.Email) (error, map[string]error) {

	_, fromDomain, err := maildomain.SplitAddress(e.From.EmailAddress)
	if err != nil {
		return err, nil
	}

	b := e.BuildBytes(ctx, true)

	to := deliveryRecipients(e)

	return doSend(ctx, fromDomain, b, to, e.From.EmailAddress)

}

func deliveryRecipients(e *parsemail.Email) []*parsemail.User {
	if e.EnvelopeTo != nil {
		to := make([]*parsemail.User, 0, len(e.EnvelopeTo))
		for _, address := range e.EnvelopeTo {
			to = append(to, &parsemail.User{EmailAddress: address})
		}
		return to
	}
	var to []*parsemail.User
	return append(append(append(to, e.To...), e.Cc...), e.Bcc...)
}

func doSend(ctx *context.Context, fromDomain string, data []byte, to []*parsemail.User, from string) (error, map[string]error) {
	if len(to) == 0 {
		return errors.New("no delivery recipients"), nil
	}
	// Validate the entire recipient list before delivering to any domain.
	for _, recipient := range to {
		if recipient == nil {
			return errors.New("invalid delivery recipient"), nil
		}
		if _, _, err := maildomain.SplitAddress(recipient.EmailAddress); err != nil {
			return errors.New("invalid delivery recipient"), nil
		}
	}

	// 按域名整理
	toByDomain := map[mxDomain][]*parsemail.User{}
	mxLookupErrors := map[mxDomain]error{}
	// mxFailoverMap 保存每个域名的完整 MX 列表（按优先级排序），用于故障转移
	mxFailoverMap := map[string][]string{}
	for _, s := range to {
		_, recipientDomain, _ := maildomain.SplitAddress(s.EmailAddress)
		// All recipient domains use normal DNS routing, including test names.
		mxInfo, lookupErr := net.LookupMX(recipientDomain)
		address := mxDomain{
			domain: recipientDomain,
			mxHost: recipientDomain,
		}
		if lookupErr != nil {
			log.WithContext(ctx).Errorf("%s 域名mx记录查询失败，检查邮箱是否存在！", s.EmailAddress)
		}
		if len(mxInfo) > 0 {
			address = mxDomain{
				domain: recipientDomain,
				mxHost: mxInfo[0].Host,
			}
			// 保存全部 MX 主机（net.LookupMX 已按优先级排序）
			allHosts := make([]string, 0, len(mxInfo))
			for _, mx := range mxInfo {
				allHosts = append(allHosts, mx.Host)
			}
			mxFailoverMap[recipientDomain] = allHosts
		}
		if lookupErr != nil {
			mxLookupErrors[address] = lookupErr
		}
		toByDomain[address] = append(toByDomain[address], s)
	}

	var errEmailAddress []string
	var errEmailAddressMu sync.Mutex

	errMap := sync.Map{}

	as := async.New(ctx)
	for domain, tos := range toByDomain {
		domain := domain
		tos := tos
		mxLookupErr := mxLookupErrors[domain]
		as.WaitProcess(func(p any) {
			recordFailure := func(err error) {
				err = deliveryFailureCause(mxLookupErr, err)
				log.WithContext(ctx).Errorf("%v 邮件投递失败%+v", tos, err)

				errEmailAddressMu.Lock()
				for _, user := range tos {
					errEmailAddress = append(errEmailAddress, user.EmailAddress)
				}
				errEmailAddressMu.Unlock()

				errMap.Store(domain.domain, err)
			}

			// 获取该域名的全部 MX 主机（按优先级排序），用于故障转移
			mxHosts := mxFailoverMap[domain.domain]
			if len(mxHosts) == 0 {
				mxHosts = []string{domain.mxHost}
			}

			// 按优先级逐个尝试 MX 主机（RFC 5321 §5.1）
			var lastErr error
			for i, mxHost := range mxHosts {
				if i > 0 {
					log.WithContext(ctx).Infof("MX %s 投递失败，尝试下一个 MX: %s (%d/%d)", mxHosts[i-1], mxHost, i+1, len(mxHosts))
				}

				err := tryDeliverToMX(ctx, mxHost, domain.domain, from, fromDomain, buildAddress(tos), data)
				if err == nil {
					return
				}
				lastErr = err

				// 5xx 永久错误是收件人/邮箱层面的拒绝，换 MX 也不会成功
				if isPermanentSMTPResponse(err) {
					recordFailure(err)
					return
				}
			}

			recordFailure(lastErr)
		}, nil)
	}
	as.Wait()

	orgMap := map[string]error{}
	errMap.Range(func(key, value any) bool {
		if value != nil {
			orgMap[key.(string)] = value.(error)
		} else {
			orgMap[key.(string)] = nil
		}

		return true
	})

	if len(errEmailAddress) > 0 {
		return errors.New("以下收件人投递失败：" + array.Join(errEmailAddress, ",")), orgMap
	}
	return nil, orgMap
}

func isPermanentSMTPResponse(err error) bool {
	var protocolErr *textproto.Error
	return errors.As(err, &protocolErr) && protocolErr.Code >= 500 && protocolErr.Code <= 599
}

// tryDeliverToMX 尝试向单个 MX 主机投递，按原有策略依次尝试
// STARTTLS:25 → 587 → 465 → 明文:25
func tryDeliverToMX(ctx *context.Context, mxHost, domain, from, fromDomain string, to []string, data []byte) error {
	// 优先尝试25端口，starttls方式投递
	err := smtp.SendMail("", mxHost+":25", nil, from, fromDomain, to, data)
	if err == nil {
		return nil
	}
	if isPermanentSMTPResponse(err) {
		return err
	}
	log.WithContext(ctx).Infof("SMTP STARTTLS on 25 Send Error. %s", err.Error())

	// 再试用587投递
	err = smtp.SendMailWithTls("", mxHost+":587", nil, from, fromDomain, to, data)
	if err == nil {
		return nil
	}
	if isPermanentSMTPResponse(err) {
		return err
	}
	log.WithContext(ctx).Infof("SMTPS on 587 Send Error. %s", err.Error())

	// 再次尝试465投递
	err = smtp.SendMailWithTls("", mxHost+":465", nil, from, fromDomain, to, data)
	if err == nil {
		return nil
	}
	if isPermanentSMTPResponse(err) {
		return err
	}
	log.WithContext(ctx).Infof("SMTPS on 465 Send Error. %s", err.Error())

	// 最后尝试非安全方式投递
	err = smtp.SendMailUnsafe("", mxHost+":25", nil, from, fromDomain, to, data)
	if err == nil {
		log.WithContext(ctx).Warnf("Send By Unsafe SMTP")
		return nil
	}

	return err
}

func deliveryFailureCause(mxLookupErr, fallbackErr error) error {
	if fallbackErr == nil || isPermanentSMTPResponse(fallbackErr) {
		return fallbackErr
	}

	var lookupDNSErr *net.DNSError
	if !errors.As(mxLookupErr, &lookupDNSErr) || (!lookupDNSErr.IsTimeout && !lookupDNSErr.IsTemporary) {
		return fallbackErr
	}

	var networkErr net.Error
	if !errors.As(fallbackErr, &networkErr) {
		return fallbackErr
	}

	return &temporaryMXFallbackError{
		lookupErr:   mxLookupErr,
		fallbackErr: fallbackErr,
	}
}

func buildAddress(u []*parsemail.User) []string {
	var ret []string

	for _, user := range u {
		ret = append(ret, user.EmailAddress)

	}

	return ret
}
