package controllers

import (
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/i18n"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/httputil"
	"github.com/Jinnrry/pmail/utils/password"
	"net/http"
)

type modifyPasswordRequest struct {
	Password string `json:"password"`
}

func ModifyPassword(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	var retData modifyPasswordRequest
	if !httputil.ReadJSON(w, req, &retData) {
		return
	}

	if retData.Password != "" {
		encodePwd := password.Encode(retData.Password)

		_, err := db.Instance.Table("user").Where("id=?", ctx.UserID).Update(map[string]interface{}{"password": encodePwd})
		if err != nil {
			response.NewErrorResponse(response.ServerError, i18n.GetText(ctx.Lang, "unknowError"), "").FPrint(w)
			return
		}

	}

	response.NewSuccessResponse(i18n.GetText(ctx.Lang, "succ")).FPrint(w)
}
