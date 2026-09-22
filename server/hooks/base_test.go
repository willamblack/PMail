package hooks

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Jinnrry/pmail/dto/parsemail"
	"github.com/Jinnrry/pmail/hooks/framework"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	log "github.com/sirupsen/logrus"
)

// fakeHook 实现 framework.EmailHook 接口，仅用于测试。
type fakeHook struct {
	name string
}

func (f *fakeHook) SendBefore(ctx *context.Context, email *parsemail.Email) {}
func (f *fakeHook) SendAfter(ctx *context.Context, email *parsemail.Email, err map[string]error) {
}
func (f *fakeHook) ReceiveParseBefore(ctx *context.Context, email *[]byte)         {}
func (f *fakeHook) ReceiveParseAfter(ctx *context.Context, email *parsemail.Email) {}
func (f *fakeHook) ReceiveSaveAfter(ctx *context.Context, email *parsemail.Email, ue []*models.UserEmail) {
}
func (f *fakeHook) GetName(ctx *context.Context) string { return f.name }
func (f *fakeHook) SettingsHtml(ctx *context.Context, url string, requestData string) string {
	return ""
}

func resetHookList() {
	HookList = map[string]framework.EmailHook{}
}

// TestRegisterAndGetHook 验证注册与按键读取使用同一个键。
// 这是历史 bug 的核心：注册用 GetName() 返回值（如 "SpamBlock"），
// 清理协程却用插件文件名（如 "spam_block"）去 delete，键不匹配导致
// 死插件永远残留在 HookList 中。
func TestRegisterAndGetHook(t *testing.T) {
	resetHookList()

	// 模拟真实流程：注册键 = GetName() 返回值。
	registeredKey := "SpamBlock"
	RegisterHook(registeredKey, &fakeHook{name: registeredKey})

	// 用同一个键能取到。
	if _, ok := GetHook(registeredKey); !ok {
		t.Fatalf("GetHook(%q) = not found, want found", registeredKey)
	}

	// 用插件文件名（不同键）取不到，说明键语义是正确的。
	if _, ok := GetHook("spam_block"); ok {
		t.Fatalf("GetHook(%q) = found, want not found (文件名不是注册键)", "spam_block")
	}

	// RemoveHook 用注册键移除后应取不到。
	RemoveHook(registeredKey)
	if _, ok := GetHook(registeredKey); ok {
		t.Fatalf("GetHook(%q) after RemoveHook = found, want not found", registeredKey)
	}
}

// TestAllHooksSnapshot 验证 AllHooks 返回的是当前快照。
func TestAllHooksSnapshot(t *testing.T) {
	resetHookList()

	RegisterHook("A", &fakeHook{name: "A"})
	RegisterHook("B", &fakeHook{name: "B"})

	snap := AllHooks()
	if len(snap) != 2 {
		t.Fatalf("AllHooks() len = %d, want 2", len(snap))
	}

	// 移除后快照不受影响（已拷贝），但新快照会变。
	RemoveHook("A")
	if len(snap) != 2 {
		t.Fatalf("snapshot should be immutable, got %d", len(snap))
	}
	if got := len(AllHooks()); got != 1 {
		t.Fatalf("AllHooks() after remove = %d, want 1", got)
	}
}

// TestHookNames 验证名称列表与注册键一致。
func TestHookNames(t *testing.T) {
	resetHookList()
	RegisterHook("SpamBlock", &fakeHook{name: "SpamBlock"})
	RegisterHook("WeChatPush", &fakeHook{name: "WeChatPush"})

	names := HookNames()
	if len(names) != 2 {
		t.Fatalf("HookNames() len = %d, want 2", len(names))
	}
	seen := map[string]bool{}
	for _, n := range names {
		seen[n] = true
	}
	if !seen["SpamBlock"] || !seen["WeChatPush"] {
		t.Fatalf("HookNames() = %v, want contain SpamBlock and WeChatPush", names)
	}
}

// TestConcurrentAccess 在 -race 下验证并发读写不会触发数据竞争。
// 模拟插件进程退出（RemoveHook）与收发邮件（AllHooks 遍历）同时进行。
func TestConcurrentAccess(t *testing.T) {
	resetHookList()
	const plugins = 50
	for i := 0; i < plugins; i++ {
		key := "hook" + string(rune('A'+i%26))
		RegisterHook(key, &fakeHook{name: key})
	}

	var wg sync.WaitGroup

	// 读侧：模拟收/发邮件时遍历插件。
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				for _, h := range AllHooks() {
					if h == nil {
						continue
					}
					_ = h.GetName(nil)
				}
			}
		}()
	}

	// 写侧：模拟插件进程退出时的移除。
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := "hook" + string(rune('A'+(i+j)%26))
				RemoveHook(key)
				RegisterHook(key, &fakeHook{name: key})
			}
		}(i)
	}

	wg.Wait()
}

type pluginLoadTestHook struct {
	onLoad func() error
}

func (h pluginLoadTestHook) Levels() []log.Level { return log.AllLevels }
func (h pluginLoadTestHook) Fire(entry *log.Entry) error {
	if entry.Message == "[short_lived] Plugin Load" {
		return h.onLoad()
	}
	return nil
}

// Exercise the actual startup/exit interleaving: the child exits while its
// GetName request is pending, then that request finishes after exit cleanup.
func TestInitDoesNotRegisterPluginAfterExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX shell and Unix socket")
	}
	t.Chdir(t.TempDir())
	if err := os.Mkdir("plugins", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("plugins/short_lived", []byte("#!/bin/sh\necho $$ > child.pid\nexec sleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	oldProcesses := processList
	processList = nil
	t.Cleanup(func() {
		Stop()
		processList = oldProcesses
	})
	oldHooks := log.StandardLogger().ReplaceHooks(make(log.LevelHooks))
	t.Cleanup(func() { log.StandardLogger().ReplaceHooks(oldHooks) })
	result := make(chan error, 1)
	var server *http.Server
	var startErr error
	log.AddHook(pluginLoadTestHook{onLoad: func() error {
		listener, err := net.Listen("unix", "plugins/1.socket")
		if err != nil {
			startErr = err
			return err
		}
		server = &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pidData, err := os.ReadFile("child.pid")
			if err != nil {
				result <- err
				return
			}
			pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
			if err != nil {
				result <- err
				return
			}
			process, err := os.FindProcess(pid)
			if err != nil {
				result <- err
				return
			}
			if err := process.Kill(); err != nil {
				result <- err
				return
			}
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				if _, err := os.Stat("plugins/1.socket"); os.IsNotExist(err) {
					result <- nil
					fmt.Fprint(w, "ShortLived")
					return
				}
				time.Sleep(time.Millisecond)
			}
			result <- fmt.Errorf("plugin exit cleanup did not remove the socket")
		})}
		go server.Serve(listener)
		return nil
	}})
	Init("test")
	if server != nil {
		defer server.Close()
	}
	if startErr != nil {
		t.Fatal(startErr)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("plugin name request was not made")
	}
	if got := HookNames(); len(got) != 0 {
		t.Fatalf("exited plugin was registered after cleanup: %v", got)
	}
}

func TestPluginRegistrationLifecycle(t *testing.T) {
	t.Run("exit before registration", func(t *testing.T) {
		resetHookList()
		registration := &hookRegistration{}
		registration.processExited()
		if registration.register("SpamBlock", &HookSender{}) || len(HookNames()) != 0 {
			t.Fatal("exited plugin was registered")
		}
	})
	t.Run("exit after registration", func(t *testing.T) {
		resetHookList()
		registration := &hookRegistration{}
		if !registration.register("SpamBlock", &HookSender{}) {
			t.Fatal("live plugin was not registered")
		}
		registration.processExited()
		registration.processExited()
		if len(HookNames()) != 0 {
			t.Fatal("exited plugin was not removed")
		}
	})
	t.Run("old exit preserves replacement", func(t *testing.T) {
		resetHookList()
		oldRegistration, replacementRegistration := &hookRegistration{}, &hookRegistration{}
		oldRegistration.register("SpamBlock", &HookSender{})
		replacement := &HookSender{}
		replacementRegistration.register("SpamBlock", replacement)
		oldRegistration.processExited()
		if current, ok := GetHook("SpamBlock"); !ok || current != replacement {
			t.Fatal("old plugin exit removed its replacement")
		}
		replacementRegistration.processExited()
		if len(HookNames()) != 0 {
			t.Fatal("replacement was not removed on exit")
		}
	})
	t.Run("concurrent registration and exit", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			resetHookList()
			registration := &hookRegistration{}
			start := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				<-start
				registration.register("SpamBlock", &HookSender{})
			}()
			go func() {
				defer wg.Done()
				<-start
				registration.processExited()
			}()
			close(start)
			wg.Wait()
			if len(HookNames()) != 0 {
				t.Fatal("concurrent exit left a dead plugin registered")
			}
		}
	})
}
