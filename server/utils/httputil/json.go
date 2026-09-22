package httputil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/Jinnrry/pmail/dto/response"
)

// MaxJSONBodySize bounds attachment uploads as well as ordinary API requests.
// Attachments are base64-encoded, so their decoded size is smaller than this.
const MaxJSONBodySize = 32 << 20

// ReadJSON rejects malformed, null and trailing JSON before a handler can mutate
// state. Unknown fields remain allowed for compatibility with existing clients.
func ReadJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, MaxJSONBodySize)
	decoder := json.NewDecoder(r.Body)
	var raw json.RawMessage
	err := decoder.Decode(&raw)
	raw = bytes.TrimSpace(raw)
	if err == nil && len(raw) > 0 && raw[0] == '{' {
		var trailing any
		if decoder.Decode(&trailing) == io.EOF && json.Unmarshal(raw, target) == nil {
			return true
		}
	}
	response.NewErrorResponse(response.ParamsError, "Invalid JSON object or request body too large", "").FPrint(w)
	return false
}

// Pagination applies the default before computing the offset and never permits
// a client to remove the database LIMIT or overflow an int on 32-bit builds.
func Pagination(page, size int) (current, limit, offset int, ok bool) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	if size > 200 {
		size = 200
	}
	if page-1 > int(^uint(0)>>1)/size {
		return 0, 0, 0, false
	}
	return page, size, (page - 1) * size, true
}
