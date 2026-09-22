package email

import (
	stdcontext "context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/dto/parsemail"
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/hooks"
	"github.com/Jinnrry/pmail/hooks/framework"
	"github.com/Jinnrry/pmail/i18n"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/async"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/httputil"
	"github.com/Jinnrry/pmail/utils/maildomain"
	"github.com/Jinnrry/pmail/utils/send"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"mime"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

type sendRequest struct {
	ReplyTo     []user       `json:"reply_to"`
	From        user         `json:"from"`
	To          []user       `json:"to"`
	Bcc         []user       `json:"bcc"`
	Cc          []user       `json:"cc"`
	Subject     string       `json:"subject"`
	Text        string       `json:"text"`   // Plaintext message (optional)
	HTML        string       `json:"html"`   // Html message (optional)
	Sender      user         `json:"sender"` // RFC Sender header metadata; does not override SMTP envelope From
	ReadReceipt []string     `json:"read_receipt"`
	Attachments []attachment `json:"attrs"`
}

type user struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type attachment struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func Send(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	var reqData sendRequest
	if !httputil.ReadJSON(w, req, &reqData) {
		return
	}

	if reqData.From.Email == "" {
		reqData.From.Email = ctx.UserAccount + "@" + config.Instance.Domain
	}
	if err := validateSendRequest(ctx, &reqData); err != nil {
		response.NewErrorResponse(response.ParamsError, err.Error(), "").FPrint(w)
		return
	}

	if reqData.From.Email == "" {
		response.NewErrorResponse(response.ParamsError, "发件人必填", "发件人必填").FPrint(w)
		return
	}

	if reqData.From.Name == "" {
		reqData.From.Name = ctx.UserName
	}

	if reqData.Subject == "" {
		response.NewErrorResponse(response.ParamsError, "邮件标题必填", "邮件标题必填").FPrint(w)
		return
	}

	if len(reqData.To)+len(reqData.Cc)+len(reqData.Bcc) == 0 {
		response.NewErrorResponse(response.ParamsError, "收件人必填", "收件人必填").FPrint(w)
		return
	}

	e := &parsemail.Email{}
	for _, reply := range reqData.ReplyTo {
		e.ReplyTo = append(e.ReplyTo, &parsemail.User{Name: reply.Name, EmailAddress: reply.Email})
	}
	e.ReadReceipt = reqData.ReadReceipt

	for _, to := range reqData.To {
		e.To = append(e.To, &parsemail.User{
			Name:         to.Name,
			EmailAddress: to.Email,
		})
	}

	for _, bcc := range reqData.Bcc {
		e.Bcc = append(e.Bcc, &parsemail.User{
			Name:         bcc.Name,
			EmailAddress: bcc.Email,
		})
	}

	for _, cc := range reqData.Cc {
		e.Cc = append(e.Cc, &parsemail.User{
			Name:         cc.Name,
			EmailAddress: cc.Email,
		})
	}

	e.From = &parsemail.User{
		Name:         reqData.From.Name,
		EmailAddress: reqData.From.Email,
	}
	if reqData.Sender.Email != "" {
		e.Sender = &parsemail.User{
			Name:         reqData.Sender.Name,
			EmailAddress: reqData.Sender.Email,
		}
	} else {
		e.Sender = &parsemail.User{
			Name:         reqData.From.Name,
			EmailAddress: reqData.From.Email,
		}
	}

	e.Text = []byte(reqData.Text)
	e.HTML = []byte(reqData.HTML)
	e.Subject = reqData.Subject
	for _, att := range reqData.Attachments {
		contentType, decoded, err := decodeAttachment(att.Data)
		if err != nil {
			log.WithContext(ctx).Errorf("附件解码错误！%v", err)
			response.NewErrorResponse(response.ParamsError, i18n.GetText(ctx.Lang, "att_err"), err.Error()).FPrint(w)
			return
		}
		e.Attachments = append(e.Attachments, &parsemail.Attachment{
			Filename:    att.Name,
			ContentType: contentType,
			Content:     decoded,
		})

	}

	log.WithContext(ctx).Debugf("插件执行--SendBefore")
	for _, hook := range hooks.AllHooks() {
		if hook == nil {
			continue
		}
		hook.SendBefore(ctx, e)
	}
	log.WithContext(ctx).Debugf("插件执行--SendBefore End")

	modelEmail := models.Email{
		Type:         1,
		Subject:      e.Subject,
		ReplyTo:      json2string(e.ReplyTo),
		FromName:     e.From.Name,
		FromAddress:  e.From.EmailAddress,
		To:           json2string(e.To),
		Bcc:          json2string(e.Bcc),
		Cc:           json2string(e.Cc),
		Text:         sql.NullString{String: string(e.Text), Valid: true},
		Html:         sql.NullString{String: string(e.HTML), Valid: true},
		Sender:       json2string(e.Sender),
		Attachments:  json2string(e.Attachments),
		SPFCheck:     1,
		DKIMCheck:    1,
		SendUserID:   ctx.UserID,
		SendDate:     time.Now(),
		CronSendTime: time.Now(),
		Status:       1,
		CreateTime:   time.Now(),
		MsgID:        parsemail.GenerateMsgID(config.Instance.Domain),
	}

	_, err := db.Instance.Insert(&modelEmail)

	if err != nil || modelEmail.Id <= 0 {
		log.WithContext(ctx).Errorf("insert email failed: %v", err)
		response.NewErrorResponse(response.ServerError, i18n.GetText(ctx.Lang, "send_fail"), "").FPrint(w)
		return
	}

	e.MessageId = cast.ToInt64(modelEmail.Id)
	e.MsgID = modelEmail.MsgID
	// Delivery outlives the HTTP response. Preserve identity/logging values but
	// do not inherit cancellation when net/http finishes the request.
	backgroundCtx := *ctx
	backgroundCtx.Context = stdcontext.WithoutCancel(ctx.Context)
	ctx = &backgroundCtx

	async.New(ctx).Process(func(p any) {
		errMsg := ""
		err, sendErr := send.Send(ctx, e)

		log.WithContext(ctx).Debugf("插件执行--SendAfter")

		as2 := async.New(ctx)
		for _, hook := range hooks.AllHooks() {
			if hook == nil {
				continue
			}
			as2.WaitProcess(func(hk any) {
				hk.(framework.EmailHook).SendAfter(ctx, e, sendErr)
			}, hook)
		}
		as2.Wait()
		log.WithContext(ctx).Debugf("插件执行--SendAfter")

		if err != nil {
			errMsg = err.Error()
			_, err := db.Instance.Exec(db.WithContext(ctx, "update email set status =2 ,error=? where id = ? "), errMsg, modelEmail.Id)
			if err != nil {
				log.WithContext(ctx).Errorf("sql Error :%+v", err)
			}

			ue := models.UserEmail{
				UserID:  ctx.UserID,
				EmailID: modelEmail.Id,
				Status:  2,
				IsRead:  1,
			}
			db.Instance.Insert(&ue)

		} else {
			_, err := db.Instance.Exec(db.WithContext(ctx, "update email set status =1  where id = ? "), modelEmail.Id)
			if err != nil {
				log.WithContext(ctx).Errorf("sql Error :%+v", err)
			}

			ue := models.UserEmail{
				UserID:  ctx.UserID,
				EmailID: modelEmail.Id,
				Status:  1,
				IsRead:  1,
			}
			db.Instance.Insert(&ue)
		}

	}, nil)

	response.NewSuccessResponse(i18n.GetText(ctx.Lang, "succ")).FPrint(w)
}

func json2string(d any) string {
	by, _ := json.Marshal(d)
	return string(by)
}

func validateSendRequest(ctx *context.Context, req *sendRequest) error {
	validateAddress := func(value *user, owned bool) error {
		if strings.ContainsAny(value.Name+value.Email, "\r\n\x00") {
			return fmt.Errorf("Invalid email header")
		}
		parsed, err := mail.ParseAddress(value.Email)
		if err != nil || parsed.Name != "" {
			return fmt.Errorf("Invalid email address")
		}
		account, domain, err := maildomain.SplitAddress(parsed.Address)
		if err != nil {
			return fmt.Errorf("Invalid email address")
		}
		if owned {
			_, allowed := maildomain.MatchRoot(domain, config.Instance.Domains, config.Instance.AcceptSubdomains)
			if !allowed || (!ctx.IsAdmin && !strings.EqualFold(account, ctx.UserAccount)) {
				return fmt.Errorf("Sender address not permitted")
			}
		}
		// Store the validated, normalized mailbox rather than the original
		// display-name/angle-bracket input; quote special local-parts correctly.
		canonical := (&mail.Address{Address: account + "@" + domain}).String()
		value.Email = strings.TrimSuffix(strings.TrimPrefix(canonical, "<"), ">")
		return nil
	}
	if err := validateAddress(&req.From, true); err != nil {
		return err
	}
	if req.Sender.Email != "" {
		if err := validateAddress(&req.Sender, true); err != nil {
			return err
		}
	}
	for _, addresses := range [][]user{req.To, req.Cc, req.Bcc, req.ReplyTo} {
		for i := range addresses {
			if err := validateAddress(&addresses[i], false); err != nil {
				return err
			}
		}
	}
	for i, address := range req.ReadReceipt {
		value := user{Email: address}
		if err := validateAddress(&value, false); err != nil {
			return err
		}
		req.ReadReceipt[i] = value.Email
	}
	if strings.ContainsAny(req.Subject, "\r\n\x00") {
		return fmt.Errorf("Invalid subject header")
	}
	for _, att := range req.Attachments {
		if strings.ContainsAny(att.Name, "\r\n\x00") {
			return fmt.Errorf("Invalid attachment name")
		}
	}
	return nil
}

func decodeAttachment(data string) (string, []byte, error) {
	if !strings.HasPrefix(data, "data:") {
		return "", nil, fmt.Errorf("Expected a base64 data URL")
	}
	mediaType, content, ok := strings.Cut(strings.TrimPrefix(data, "data:"), ";base64,")
	if !ok || strings.ContainsAny(mediaType, "\r\n\x00") {
		return "", nil, fmt.Errorf("Invalid attachment data URL")
	}
	if _, _, err := mime.ParseMediaType(mediaType); err != nil {
		return "", nil, fmt.Errorf("Invalid attachment media type")
	}
	decoded, err := base64.StdEncoding.DecodeString(content)
	return mediaType, decoded, err
}
