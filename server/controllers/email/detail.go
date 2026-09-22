package email

import (
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/services/detail"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/httputil"
	"net/http"
)

type emailDetailRequest struct {
	ID int `json:"id"`
}

func EmailDetail(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	var retData emailDetailRequest
	if !httputil.ReadJSON(w, req, &retData) {
		return
	}

	if retData.ID <= 0 {
		response.NewErrorResponse(response.ParamsError, "ID错误", "").FPrint(w)
		return
	}

	email, err := detail.GetEmailDetail(ctx, retData.ID, true)
	if err != nil {
		response.NewErrorResponse(response.ServerError, err.Error(), "").FPrint(w)
		return
	}

	response.NewSuccessResponse(email).FPrint(w)

}
