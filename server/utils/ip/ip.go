package ip

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

var ip string
var ipMu sync.Mutex

func GetIp() string {
	ipMu.Lock()
	defer ipMu.Unlock()
	if ip != "" {
		return ip
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://ip-api.com/json/?lang=zh-CN")
	if err != nil {
		return "[Your Server IP]"
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if err == nil {
			var queryRes map[string]string
			_ = json.Unmarshal(body, &queryRes)
			ip = queryRes["query"]
			return queryRes["query"]
		}
	}
	return "[Your Server IP]"
}
