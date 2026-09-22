package smtp_server

import (
	"database/sql"
	"errors"
	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/id"
	"github.com/Jinnrry/pmail/utils/maildomain"
	"github.com/Jinnrry/pmail/utils/password"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
)

// The Backend implements SMTP server methods.
type Backend struct{}

func (bkd *Backend) NewSession(conn *smtp.Conn) (smtp.Session, error) {

	remoteAddress := conn.Conn().RemoteAddr()
	ctx := &context.Context{}
	ctx.SetValue(context.LogID, id.GenLogID())
	log.WithContext(ctx).Debugf("新SMTP连接")

	return &Session{
		RemoteAddress: remoteAddress,
		Helo:          conn.Hostname(),
		Ctx:           ctx,
	}, nil
}

// A Session is returned after EHLO.
type Session struct {
	RemoteAddress net.Addr
	Helo          string
	User          string
	From          string
	To            []string
	Ctx           *context.Context
}

// AuthMechanisms returns a slice of available auth mechanisms
// supported in this example.
func (s *Session) AuthMechanisms() []string {
	return []string{sasl.Plain, sasl.Login}
}

// Auth is the handler for supported authenticators.
func (s *Session) Auth(mech string) (sasl.Server, error) {
	log.WithContext(s.Ctx).Debugf("Auth :%s", mech)
	if mech == sasl.Plain {
		return sasl.NewPlainServer(func(identity, username, password string) error {
			return s.AuthPlain(username, password)
		}), nil
	}

	if mech == sasl.Login {
		return NewLoginServer(func(username, password string) error {
			return s.AuthPlain(username, password)
		}), nil
	}

	return nil, errors.New("Auth Not Supported")
}

func (s *Session) AuthPlain(username, pwd string) error {
	log.WithContext(s.Ctx).Debugf("Auth %s", username)

	s.User = username

	var user models.User

	encodePwd := password.Encode(pwd)

	if strings.Contains(username, "@") {
		account, domain, err := maildomain.SplitAddress(username)
		if err != nil {
			return errors.New("password error")
		}
		if _, ok := maildomain.MatchRoot(domain, config.Instance.Domains, config.Instance.AcceptSubdomains); !ok {
			return errors.New("password error")
		}
		username = account
	}

	_, err := db.Instance.Where("account =? and password =? and disabled=0", username, encodePwd).Get(&user)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("%+v", err)
	}

	if user.ID > 0 {
		s.Ctx.UserAccount = user.Account
		s.Ctx.UserID = user.ID
		s.Ctx.UserName = user.Name
		s.Ctx.IsAdmin = user.IsAdmin == 1

		log.WithContext(s.Ctx).Debugf("Auth Success user_id=%d", user.ID)
		return nil
	}

	log.WithContext(s.Ctx).Debugf("登陆错误 %s", username)
	return errors.New("password error")
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	if s.Ctx.UserID > 0 && !s.senderAuthorized(from) {
		return senderRejectedSMTPError()
	}
	log.WithContext(s.Ctx).Debugf("Mail Success %+v %+v", from, opts)
	s.From = from
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	if s.Ctx.UserID == 0 {
		_, domain, err := maildomain.SplitAddress(to)
		if err != nil {
			return badRecipientSMTPError()
		}
		if _, ok := maildomain.MatchRoot(domain, config.Instance.Domains, config.Instance.AcceptSubdomains); !ok {
			return relayDeniedSMTPError()
		}
	}
	log.WithContext(s.Ctx).Debugf("Rcpt Success %+v", to)

	s.To = append(s.To, to)
	return nil
}

func (s *Session) Reset() {
	s.From = ""
	s.To = nil
}

func (s *Session) senderAuthorized(address string) bool {
	account, domain, err := maildomain.SplitAddress(address)
	if err != nil {
		return false
	}
	if _, ok := maildomain.MatchRoot(domain, config.Instance.Domains, config.Instance.AcceptSubdomains); !ok {
		return false
	}
	return s.Ctx.IsAdmin || strings.EqualFold(account, s.Ctx.UserAccount)
}

func badRecipientSMTPError() error {
	return &smtp.SMTPError{
		Code:         501,
		EnhancedCode: smtp.EnhancedCode{5, 1, 3},
		Message:      "Bad recipient address syntax",
	}
}

func relayDeniedSMTPError() error {
	return &smtp.SMTPError{
		Code:         550,
		EnhancedCode: smtp.EnhancedCode{5, 7, 1},
		Message:      "Relay denied",
	}
}

func senderRejectedSMTPError() error {
	return &smtp.SMTPError{
		Code:         553,
		EnhancedCode: smtp.EnhancedCode{5, 7, 1},
		Message:      "Sender address is not authorized",
	}
}

func (s *Session) Logout() error {
	return nil
}
