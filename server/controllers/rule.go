package controllers

import (
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/dto"
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/i18n"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/services/rule"
	"github.com/Jinnrry/pmail/services/rule/match"
	"github.com/Jinnrry/pmail/utils/address"
	"github.com/Jinnrry/pmail/utils/array"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/errors"
	"github.com/Jinnrry/pmail/utils/httputil"
	"net/http"
)

func GetRule(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	res := rule.GetAllRules(ctx, ctx.UserID)
	response.NewSuccessResponse(res).FPrint(w)
}

func UpsertRule(ctx *context.Context, w http.ResponseWriter, req *http.Request) {

	var data *dto.Rule
	if !httputil.ReadJSON(w, req, &data) {
		return
	}
	if data.Action < dto.READ || data.Action > dto.MOVE {
		response.NewErrorResponse(response.ParamsError, "Invalid rule action", "").FPrint(w)
		return
	}

	if data.Action == dto.FORWARD && !address.IsValidEmailAddress(data.Params) {

		response.NewErrorResponse(response.ParamsError, "ParamsError error", i18n.GetText(ctx.Lang, "invalid_email_address")).FPrint(w)
		return
	}

	for _, r := range data.Rules {
		if r == nil || !array.InArray(r.Field, []string{"From", "Subject", "To", "Cc", "Text", "Html", "Content"}) ||
			!array.InArray(r.Type, []string{match.RuleTypeRegex, match.RuleTypeContains, match.RuleTypeEq}) {
			response.NewErrorResponse(response.ParamsError, "ParamsError error", "params error! Rule Field Error!").FPrint(w)
			return
		}
		if r.Type == match.RuleTypeRegex {
			if err := match.ValidateRegex(r.Rule); err != nil {
				response.NewErrorResponse(response.ParamsError, "Invalid regular expression", "").FPrint(w)
				return
			}
		}
	}

	err := save(ctx, data.Encode())
	if err != nil {
		response.NewErrorResponse(response.ServerError, "server error", err).FPrint(w)
		return
	}
	response.NewSuccessResponse("succ").FPrint(w)
}

func save(ctx *context.Context, p *models.Rule) error {

	if p.Id > 0 {
		_, err := db.Instance.Exec(db.WithContext(ctx, "update rule set name=? ,value = ? ,action = ?,params = ?,sort = ? where id = ? and user_id = ?"), p.Name, p.Value, p.Action, p.Params, p.Sort, p.Id, ctx.UserID)
		if err != nil {
			return errors.Wrap(err)
		}
		return nil
	} else {
		_, err := db.Instance.Exec(db.WithContext(ctx, "insert into rule (name,value,user_id,action,params,sort) values (?,?,?,?,?,?)"), p.Name, p.Value, ctx.UserID, p.Action, p.Params, p.Sort)
		if err != nil {
			return errors.Wrap(err)
		}
		return nil
	}

}

type delRuleReq struct {
	Id int `json:"id"`
}

func DelRule(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	var data delRuleReq
	if !httputil.ReadJSON(w, req, &data) {
		return
	}

	if data.Id <= 0 {
		response.NewErrorResponse(response.ParamsError, "params error", "id is empty").FPrint(w)
		return
	}

	_, err := db.Instance.Exec(db.WithContext(ctx, "delete from rule where id =? and user_id =?"), data.Id, ctx.UserID)
	if err != nil {
		response.NewErrorResponse(response.ServerError, "unknown error", err).FPrint(w)
		return
	}

	response.NewSuccessResponse("succ").FPrint(w)
}
