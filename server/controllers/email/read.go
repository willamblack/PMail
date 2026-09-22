package email

import (
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/services/detail"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/httputil"
	"net/http"
)

type markReadRequest struct {
	IDs []int `json:"ids"`
}

func MarkRead(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	var reqData markReadRequest
	if !httputil.ReadJSON(w, req, &reqData) {
		return
	}

	if len(reqData.IDs) <= 0 {
		response.NewErrorResponse(response.ParamsError, "ID错误", "").FPrint(w)
		return
	}

	for _, id := range reqData.IDs {
		if _, err := detail.GetEmailDetail(ctx, id, true); err != nil {
			response.NewErrorResponse(response.ServerError, err.Error(), "").FPrint(w)
			return
		}
	}
	response.NewSuccessResponse("success").FPrint(w)

}
