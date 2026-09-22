package hooks

import (
	oContext "context"
	"encoding/json"
	"fmt"
	"github.com/Jinnrry/pmail/dto/parsemail"
	"github.com/Jinnrry/pmail/hooks/framework"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// HookList
var HookList map[string]framework.EmailHook

// hookListMu 保护 HookList。插件进程退出时会并发地从 HookList 移除条目，
// 而收发邮件的 goroutine 会并发读取，必须加锁避免数据竞争。
var hookListMu sync.RWMutex

// AllHooks 返回当前已注册插件的快照，供收发邮件时并发安全地遍历。
func AllHooks() []framework.EmailHook {
	hookListMu.RLock()
	defer hookListMu.RUnlock()
	list := make([]framework.EmailHook, 0, len(HookList))
	for _, h := range HookList {
		list = append(list, h)
	}
	return list
}

// HookNames 返回已注册插件的名称列表。
func HookNames() []string {
	hookListMu.RLock()
	defer hookListMu.RUnlock()
	names := make([]string, 0, len(HookList))
	for name := range HookList {
		names = append(names, name)
	}
	return names
}

// GetHook 返回指定名称的插件。
func GetHook(name string) (framework.EmailHook, bool) {
	hookListMu.RLock()
	defer hookListMu.RUnlock()
	h, ok := HookList[name]
	return h, ok
}

// RegisterHook 注册一个插件。
func RegisterHook(name string, h framework.EmailHook) {
	hookListMu.Lock()
	defer hookListMu.Unlock()
	HookList[name] = h
}

// RemoveHook 移除指定名称的插件。
func RemoveHook(name string) {
	hookListMu.Lock()
	defer hookListMu.Unlock()
	delete(HookList, name)
}

type HookSender struct {
	httpc  http.Client
	name   string
	socket string
}

func (h *HookSender) ReceiveSaveAfter(ctx *context.Context, email *parsemail.Email, ue []*models.UserEmail) {
	log.WithContext(ctx).Debugf("[%s]Plugin ReceiveSaveAfter Start", h.name)

	dto := framework.HookDTO{
		Ctx:       ctx,
		Email:     email,
		UserEmail: ue,
	}
	body, _ := json.Marshal(dto)

	ret, err := h.httpc.Post("http://unix/ReceiveSaveAfter", "application/json", strings.NewReader(string(body)))
	if err != nil {
		log.WithContext(ctx).Errorf("[%s] Error! %v", h.name, err)
		return
	}
	defer ret.Body.Close()
	_, _ = io.Copy(io.Discard, ret.Body)

	log.WithContext(ctx).Debugf("[%s]Plugin ReceiveSaveAfter End", h.name)
}

func (h *HookSender) SendBefore(ctx *context.Context, email *parsemail.Email) {
	log.WithContext(ctx).Debugf("[%s]Plugin SendBefore Start", h.name)

	dto := framework.HookDTO{
		Ctx:   ctx,
		Email: email,
	}
	body, _ := json.Marshal(dto)

	ret, err := h.httpc.Post("http://unix/SendBefore", "application/json", strings.NewReader(string(body)))
	if err != nil {
		log.WithContext(ctx).Errorf("[%s] Error! %v", h.name, err)
		return
	}

	defer ret.Body.Close()
	body, _ = io.ReadAll(ret.Body)
	json.Unmarshal(body, &dto)

	ctx = dto.Ctx
	email = dto.Email
	log.WithContext(ctx).Debugf("[%s]Plugin SendBefore End", h.name)

}

func (h *HookSender) SendAfter(ctx *context.Context, email *parsemail.Email, err map[string]error) {
	log.WithContext(ctx).Debugf("[%s]Plugin SendAfter Start", h.name)
	dto := framework.HookDTO{
		Ctx:    ctx,
		Email:  email,
		ErrMap: err,
	}
	body, _ := json.Marshal(dto)

	ret, errL := h.httpc.Post("http://unix/SendAfter", "application/json", strings.NewReader(string(body)))
	if errL != nil {
		log.WithContext(ctx).Errorf("[%s] Error! %v", h.name, errL)
		return
	}

	log.WithContext(ctx).Debugf("[%s]Plugin SendAfter End", h.name)
	defer ret.Body.Close()
	_, _ = io.Copy(io.Discard, ret.Body)

}

func (h *HookSender) ReceiveParseBefore(ctx *context.Context, email *[]byte) {
	log.WithContext(ctx).Debugf("[%s]Plugin ReceiveParseBefore Start", h.name)

	dto := framework.HookDTO{
		Ctx:       ctx,
		EmailByte: email,
	}
	body, _ := json.Marshal(dto)

	ret, errL := h.httpc.Post("http://unix/ReceiveParseBefore", "application/json", strings.NewReader(string(body)))
	if errL != nil {
		log.WithContext(ctx).Errorf("[%s] Error! %v", h.name, errL)
		return
	}

	defer ret.Body.Close()
	body, _ = io.ReadAll(ret.Body)
	json.Unmarshal(body, &dto)

	ctx = dto.Ctx
	email = dto.EmailByte
	log.WithContext(ctx).Debugf("[%s]Plugin ReceiveParseBefore End", h.name)

}

func (h *HookSender) ReceiveParseAfter(ctx *context.Context, email *parsemail.Email) {
	log.WithContext(ctx).Debugf("[%s]Plugin ReceiveParseAfter Start", h.name)

	dto := framework.HookDTO{
		Ctx:   ctx,
		Email: email,
	}
	body, _ := json.Marshal(dto)

	ret, errL := h.httpc.Post("http://unix/ReceiveParseAfter", "application/json", strings.NewReader(string(body)))
	if errL != nil {
		log.WithContext(ctx).Errorf("[%s] Error! %v", h.name, errL)
		return
	}

	defer ret.Body.Close()
	body, _ = io.ReadAll(ret.Body)
	json.Unmarshal(body, &dto)

	ctx = dto.Ctx
	email = dto.Email
	log.WithContext(ctx).Debugf("[%s]Plugin ReceiveParseAfter End", h.name)

}

// GetName 获取插件名称
func (h *HookSender) GetName(ctx *context.Context) string {

	dto := framework.HookDTO{
		Ctx: ctx,
	}
	body, _ := json.Marshal(dto)

	ret, errL := h.httpc.Post("http://unix/GetName", "application/json", strings.NewReader(string(body)))
	if errL != nil {
		log.WithContext(ctx).Errorf("[%s] Error! %v", h.name, errL)
		return ""
	}

	defer ret.Body.Close()
	body, _ = io.ReadAll(ret.Body)

	return string(body)
}

// SettingsHtml 插件页面
func (h *HookSender) SettingsHtml(ctx *context.Context, url string, requestData string) string {

	dto := framework.SettingsHtmlRequest{
		Ctx:         ctx,
		URL:         url,
		RequestData: requestData,
	}
	body, _ := json.Marshal(dto)

	ret, errL := h.httpc.Post("http://unix/SettingsHtml", "application/json", strings.NewReader(string(body)))
	if errL != nil {
		log.WithContext(ctx).Errorf("[%s] Error! %v", h.name, errL)
		return ""
	}

	defer ret.Body.Close()
	body, _ = io.ReadAll(ret.Body)

	return string(body)

}

func NewHookSender(socketPath string, name string, serverVersion string) *HookSender {
	httpc := http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			DialContext: func(ctx oContext.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
	return &HookSender{
		httpc:  httpc,
		socket: socketPath,
		name:   name,
	}
}

var processList []*os.Process

// hookRegistration serializes registration with process exit. Protecting only
// the name is insufficient: an exited process must never be registered later.
type hookRegistration struct {
	mu     sync.Mutex
	name   string
	hook   *HookSender
	exited bool
}

func (r *hookRegistration) register(name string, hook *HookSender) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.exited {
		return false
	}
	RegisterHook(name, hook)
	r.name, r.hook = name, hook
	return true
}

func (r *hookRegistration) processExited() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exited = true
	if r.hook == nil {
		return
	}
	hookListMu.Lock()
	defer hookListMu.Unlock()
	// Another process may have registered the same name in the meantime.
	// Only remove this process's instance, never its replacement.
	if current, ok := HookList[r.name].(*HookSender); ok && current == r.hook {
		delete(HookList, r.name)
	}
}

// Init 注册hook对象
func Init(serverVersion string) {

	HookList = map[string]framework.EmailHook{}
	env := os.Environ()
	procAttr := &os.ProcAttr{
		Env: env,
		Files: []*os.File{
			os.Stdin,
			os.Stdout,
			os.Stderr,
		},
	}

	root := "./plugins"

	pluginNo := 1
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() && (!strings.Contains(info.Name(), ".") || strings.Contains(info.Name(), ".exe")) {

			socketPath := fmt.Sprintf("%s/%d.socket", root, pluginNo)

			os.Remove(socketPath)

			//socketPath = "/PMail/server/hooks/spam_block/1555.socket"  //debug

			log.Infof("[%s] Plugin Load", info.Name())
			p, err := os.StartProcess(path, []string{
				info.Name(),
				fmt.Sprintf("%d.socket", pluginNo),
			}, procAttr)
			if err != nil {
				log.Errorf("Plug Load Error! %v", err)
				return nil
			}
			fmt.Printf("[%s] Plugin Start! PID:%d", info.Name(), p.Pid)
			processList = append(processList, p)

			pluginNo++

			registration := &hookRegistration{}

			go func() {
				stat, err := p.Wait()
				registration.processExited()
				log.Errorf("[%s] Plugin Stop. Error:%v Stat:%v", info.Name(), err, stat)
				os.Remove(socketPath)
			}()

			loadSucc := false
			for i := 0; i < 5; i++ {
				time.Sleep(1 * time.Second)
				if _, err := os.Stat(socketPath); err == nil {
					loadSucc = true
					break
				}
				if i == 4 {
					log.Errorf("[%s] Start Fail!", info.Name())
				}
			}
			if loadSucc {
				hk := NewHookSender(socketPath, info.Name(), serverVersion)
				hkName := hk.GetName(&context.Context{})
				if hkName == "" {
					// GetName 失败时退回插件文件名作为键，避免出现空键。
					hkName = info.Name()
				}
				if registration.register(hkName, hk) {
					log.Infof("[%s] Plugin Load Success!", hkName)
				}
			}

		}

		return nil
	})

}

func Stop() {
	log.Info("Plugin Stop")
	for _, process := range processList {
		process.Kill()
	}
}
