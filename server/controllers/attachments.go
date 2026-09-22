package controllers

import (
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/services/attachments"
	"github.com/Jinnrry/pmail/utils/context"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func GetAttachments(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	urlInfos := strings.Split(req.URL.EscapedPath(), "/")
	if len(urlInfos) != 4 {
		response.NewErrorResponse(response.ParamsError, "", "").FPrint(w)
		return
	}
	emailId, err := strconv.Atoi(urlInfos[2])
	if err != nil || emailId <= 0 {
		http.NotFound(w, req)
		return
	}
	cid, err := url.PathUnescape(urlInfos[3])
	if err != nil || cid == "" {
		http.NotFound(w, req)
		return
	}

	contentType, content := attachments.GetAttachments(ctx, emailId, cid)

	if len(content) == 0 {
		response.NewErrorResponse(response.ParamsError, "", "").FPrint(w)
		return
	}
	setAttachmentHeaders(w, contentType, "attachment", true)
	w.Write(content)
}

func Download(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	urlInfos := strings.Split(req.URL.Path, "/")
	if len(urlInfos) != 5 {
		response.NewErrorResponse(response.ParamsError, "", "").FPrint(w)
		return
	}
	emailId, err := strconv.Atoi(urlInfos[3])
	index, indexErr := strconv.Atoi(urlInfos[4])
	if err != nil || indexErr != nil || emailId <= 0 || index < 0 {
		http.NotFound(w, req)
		return
	}

	fileName, content := attachments.GetAttachmentsByIndex(ctx, emailId, index)

	if len(content) == 0 {
		response.NewErrorResponse(response.ParamsError, "", "").FPrint(w)
		return
	}
	setAttachmentHeaders(w, "application/octet-stream", fileName, false)
	w.Write(content)
}

// Inline mail parts are untrusted. Only passive image formats may render on the
// application origin; HTML/SVG/XML and unknown types must download, not execute.
func setAttachmentHeaders(w http.ResponseWriter, contentType, filename string, inline bool) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Cache-Control", "private, no-store")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && inline {
		switch mediaType {
		case "image/png", "image/jpeg", "image/gif", "image/webp", "image/avif", "image/bmp":
			w.Header().Set("Content-Type", mediaType)
			return
		}
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	filename = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, filename)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
}
