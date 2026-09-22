package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/Jinnrry/pmail/dto/parsemail"
	"github.com/Jinnrry/pmail/hooks/framework"
	"github.com/Jinnrry/pmail/hooks/spam_block/tools"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"html"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type SpamBlock struct {
	mu  sync.RWMutex
	cfg SpamBlockConfig
	hc  *http.Client
}

func (s *SpamBlock) SendBefore(ctx *context.Context, email *parsemail.Email) {

}

func (s *SpamBlock) SendAfter(ctx *context.Context, email *parsemail.Email, err map[string]error) {

}

func (s *SpamBlock) ReceiveParseBefore(ctx *context.Context, email *[]byte) {

}

// GetName 获取插件名称
func (s *SpamBlock) GetName(ctx *context.Context) string {
	return "SpamBlock"
}

//go:embed static/index.html
var index string

//go:embed static/jquery.js
var jquery string

// SettingsHtml 插件页面
func (s *SpamBlock) SettingsHtml(ctx *context.Context, url string, requestData string) string {

	if strings.Contains(url, "jquery.js") {
		return jquery
	}

	if strings.Contains(url, "index.html") {
		if !ctx.IsAdmin {
			return fmt.Sprintf(`
<div>
	Please contact the administrator for configuration.
</div>
`)
		}
		s.mu.RLock()
		cfg := s.cfg
		s.mu.RUnlock()
		return fmt.Sprintf(index, html.EscapeString(cfg.ApiURL), cfg.ApiTimeout, cfg.Threshold)
	}
	if !ctx.IsAdmin {
		return "No Access Privileges"
	}

	var cfg SpamBlockConfig
	var tempCfg map[string]string
	err := json.Unmarshal([]byte(requestData), &tempCfg)
	if err != nil {
		return err.Error()
	}
	cfg.ApiURL = tempCfg["url"]
	cfg.Threshold = cast.ToFloat64(tempCfg["threshold"])
	cfg.ApiTimeout = cast.ToInt(tempCfg["timeout"])
	if cfg.ApiTimeout <= 0 || cfg.ApiTimeout > 30000 || cfg.Threshold < 0 || cfg.Threshold > 100 {
		return "invalid timeout or threshold"
	}
	err = saveConfig(cfg)
	if err != nil {
		return err.Error()
	}

	s.mu.Lock()
	s.cfg = cfg
	s.hc = &http.Client{Timeout: time.Duration(cfg.ApiTimeout) * time.Millisecond}
	s.mu.Unlock()

	return "success"

}

type ModelResponse struct {
	Predictions [][]float64 `json:"predictions"`
}

type ApiRequest struct {
	Instances []InstanceItem `json:"instances"`
}

type InstanceItem struct {
	Token []string `json:"token"`
}

func (s *SpamBlock) ReceiveParseAfter(ctx *context.Context, email *parsemail.Email) {
	s.mu.RLock()
	cfg, client := s.cfg, s.hc
	s.mu.RUnlock()
	if cfg.ApiURL == "" || client == nil {
		return
	}

	content := tools.Trim(tools.TrimHtml(string(email.HTML)))
	if content == "" {
		content = tools.Trim(string(email.Text))
	}

	reqData := ApiRequest{
		Instances: []InstanceItem{
			{
				Token: []string{
					fmt.Sprintf("%s %s", email.Subject, content),
				},
			},
		},
	}

	str, _ := json.Marshal(reqData)

	resp, err := client.Post(cfg.ApiURL, "application/json", strings.NewReader(string(str)))
	if err != nil {
		log.Errorf("API Error: %v", err)
		return
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))

	modelResponse := ModelResponse{}
	err = json.Unmarshal(body, &modelResponse)
	if err != nil {
		log.WithContext(ctx).Errorf("API Error: %v", err)
		return
	}

	if len(modelResponse.Predictions) == 0 {
		log.WithContext(ctx).Errorf("API Response Error: %v", string(body))
		return
	}

	classes := modelResponse.Predictions[0]
	if len(classes) != 3 {
		return
	}
	var maxScore float64
	var maxClass int
	for i, score := range classes {
		if score > maxScore {
			maxScore = score
			maxClass = i
		}
	}

	switch maxClass {
	case 0:
		log.WithContext(ctx).Infof("[Spam Check Result: %f Normal] %s", maxScore, email.Subject)
	case 1:
		log.WithContext(ctx).Infof("[Spam Check Result: %f Spam ] %s", maxScore, email.Subject)
	case 2:
		log.WithContext(ctx).Infof("[Spam Check Result: %f Blackmail ] %s", maxScore, email.Subject)
	}

	if maxClass != 0 && maxScore > cfg.Threshold/100 {
		if maxClass == 2 {
			email.Status = 3
		} else {
			email.Status = 5
		}
	}
}

func (s *SpamBlock) ReceiveSaveAfter(ctx *context.Context, email *parsemail.Email, ue []*models.UserEmail) {

}

type SpamBlockConfig struct {
	ApiURL     string  `json:"apiURL"`
	ApiTimeout int     `json:"apiTimeout"` // 单位毫秒
	Threshold  float64 `json:"threshold"`
}

func NewSpamBlockHook() *SpamBlock {

	var pluginConfig SpamBlockConfig
	if _, err := os.Stat("./plugins/spam_block_config.json"); err == nil {
		cfgData, err := os.ReadFile("./plugins/spam_block_config.json")
		if err == nil {
			json.Unmarshal(cfgData, &pluginConfig)
		}
	}

	log.Infof("Config: %+v", pluginConfig)

	if pluginConfig.ApiTimeout == 0 {
		pluginConfig.ApiTimeout = 3000
	}

	if pluginConfig.Threshold == 0 {
		pluginConfig.Threshold = 20
	}

	hc := &http.Client{
		Timeout: time.Duration(pluginConfig.ApiTimeout) * time.Millisecond,
	}

	return &SpamBlock{
		cfg: pluginConfig,
		hc:  hc,
	}
}

func saveConfig(cfg SpamBlockConfig) error {
	data, _ := json.Marshal(cfg)
	err := os.WriteFile("./plugins/spam_block_config.json", data, 0777)
	return err
}

func main() {
	log.Infof("SpamBlockPlug Star Success")
	instance := NewSpamBlockHook()
	if instance == nil {
		return
	}
	framework.CreatePlugin("spam_block", instance).Run()
}
