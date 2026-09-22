package controllers

import (
	"database/sql"
	"encoding/json"
	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/i18n"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/session"
	"github.com/Jinnrry/pmail/utils/array"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/errors"
	"github.com/Jinnrry/pmail/utils/httputil"
	"github.com/Jinnrry/pmail/utils/password"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type loginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

func Login(ctx *context.Context, w http.ResponseWriter, req *http.Request) {

	var reqData loginRequest
	if !httputil.ReadJSON(w, req, &reqData) {
		return
	}

	var user models.User

	encodePwd := password.Encode(reqData.Password)
	_, err := db.Instance.Where("account =? and password =? and disabled=0", reqData.Account, encodePwd).Get(&user)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Errorf("%+v", err)
	}

	if user.ID != 0 {
		if err := session.Instance.RenewToken(req.Context()); err != nil {
			response.NewErrorResponse(response.ServerError, "Unable to establish session", "").FPrint(w)
			return
		}
		userStr, _ := json.Marshal(user)
		session.Instance.Put(req.Context(), "user", string(userStr))
		session.Instance.Put(req.Context(), "credential_version", session.CredentialVersion(user.Password))

		domains := config.Instance.Domains
		domains = array.Difference(domains, []string{config.Instance.Domain})
		domains = append([]string{config.Instance.Domain}, domains...)

		response.NewSuccessResponse(map[string]any{
			"account":  user.Account,
			"name":     user.Name,
			"is_admin": user.IsAdmin,
			"domains":  domains,
		}).FPrint(w)
	} else {
		response.NewErrorResponse(response.ParamsError, i18n.GetText(ctx.Lang, "aperror"), "").FPrint(w)
	}
}

func Logout(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	if err := session.Instance.Destroy(ctx.Context); err != nil {
		response.NewErrorResponse(response.ServerError, "Unable to end session", "").FPrint(w)
		return
	}
	response.NewSuccessResponse("Success").FPrint(w)
}
