package email

import (
	"encoding/json"
	"github.com/Jinnrry/pmail/dto"
	"github.com/Jinnrry/pmail/dto/parsemail"
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/services/list"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/httputil"
	"github.com/spf13/cast"
	"math"
	"net/http"
	"strings"
)

type emailListResponse struct {
	CurrentPage int         `json:"current_page"`
	TotalPage   int         `json:"total_page"`
	List        []*emilItem `json:"list"`
}

type emilItem struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Desc       string `json:"desc"`
	Datetime   string `json:"datetime"`
	IsRead     bool   `json:"is_read"`
	Sender     User   `json:"sender"`
	To         []User `json:"to"`
	Recipients []User `json:"recipients"`
	Dangerous  bool   `json:"dangerous"`
	Error      string `json:"error"`
}

type User struct {
	Name         string `json:"Name"`
	EmailAddress string `json:"EmailAddress"`
}

type emailRequest struct {
	Keyword     string `json:"keyword"`
	SearchField string `json:"search_field"`
	Tag         string `json:"tag"`
	CurrentPage int    `json:"current_page"`
	PageSize    int    `json:"page_size"`
}

func EmailList(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	lst := make([]*emilItem, 0)
	var retData emailRequest
	if !httputil.ReadJSON(w, req, &retData) {
		return
	}
	page, limit, offset, ok := httputil.Pagination(retData.CurrentPage, retData.PageSize)
	if !ok {
		response.NewErrorResponse(response.ParamsError, "Invalid pagination", "").FPrint(w)
		return
	}
	retData.CurrentPage, retData.PageSize = page, limit

	var tagInfo dto.SearchTag = dto.SearchTag{
		Type:    -1,
		Status:  -1,
		GroupId: -1,
	}
	_ = json.Unmarshal([]byte(retData.Tag), &tagInfo)

	emailList, total := list.SearchEmailList(ctx, tagInfo, retData.Keyword, retData.SearchField, false, offset, retData.PageSize)

	for _, email := range emailList {
		var sender User
		_ = json.Unmarshal([]byte(email.Sender), &sender)

		if sender.EmailAddress == "" {
			sender.EmailAddress = email.FromAddress
			sender.Name = email.FromName
		}

		var tos []User
		_ = json.Unmarshal([]byte(email.To), &tos)
		recipients := collectRecipients(email.To, email.Cc, email.Bcc)
		authentication := parsemail.NewEmailAuthentication(email.SPFCheck == 1, email.DKIMCheck == 1)

		lst = append(lst, &emilItem{
			ID:         email.Id,
			Title:      email.Subject,
			Desc:       email.Text.String,
			Datetime:   email.SendDate.Format("2006-01-02 15:04:05"),
			IsRead:     email.IsRead == 1,
			Sender:     sender,
			To:         tos,
			Recipients: recipients,
			Dangerous:  authentication.Dangerous,
			Error:      email.Error.String,
		})
	}

	ret := emailListResponse{
		CurrentPage: retData.CurrentPage,
		TotalPage:   cast.ToInt(math.Ceil(cast.ToFloat64(total) / cast.ToFloat64(retData.PageSize))),
		List:        lst,
	}
	response.NewSuccessResponse(ret).FPrint(w)
}

func collectRecipients(values ...string) []User {
	var recipients []User
	seen := make(map[string]struct{})
	for _, value := range values {
		var users []User
		if err := json.Unmarshal([]byte(value), &users); err != nil {
			continue
		}
		for _, user := range users {
			address := strings.TrimSpace(user.EmailAddress)
			key := strings.ToLower(address)
			if key == "" {
				key = "name:" + strings.ToLower(strings.TrimSpace(user.Name))
			}
			if key == "name:" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			recipients = append(recipients, user)
		}
	}
	return recipients
}
