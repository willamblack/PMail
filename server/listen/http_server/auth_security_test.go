package http_server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/controllers"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/session"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/password"
	"github.com/alexedwards/scs/v2"
	_ "modernc.org/sqlite"
	"xorm.io/xorm"
)

func setupHTTPAuthDB(t *testing.T) {
	t.Helper()
	oldDB, oldConfig, oldInit, oldSession := db.Instance, config.Instance, config.IsInit, session.Instance
	engine, err := xorm.NewEngine("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	engine.SetMaxOpenConns(1)
	if err = engine.Sync2(&models.User{}); err != nil {
		t.Fatal(err)
	}
	db.Instance = engine
	config.Instance = &config.Config{Domain: "example.com", Domains: []string{"example.com"}, WebDomain: "mail.example.com"}
	config.IsInit = true
	session.Instance = scs.New()
	_, err = engine.Insert(&models.User{ID: 1, Account: "admin", Name: "Admin", Password: password.Encode("secret"), IsAdmin: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = engine.Close()
		db.Instance = oldDB
		config.Instance = oldConfig
		config.IsInit = oldInit
		session.Instance = oldSession
	})
}

func TestTokenRejectsFutureAndMalformedTimestamp(t *testing.T) {
	setupHTTPAuthDB(t)
	for _, ts := range []string{"not-a-number", fmt.Sprint(time.Now().Unix() + 3600), fmt.Sprint(time.Now().Unix() - 6)} {
		token := "admin:" + password.Md5Encode(password.Encode("secret")+ts) + ":" + ts
		if _, err := getLoginInfoByToken(token); err == nil {
			t.Fatalf("timestamp %q accepted", ts)
		}
	}
	ts := fmt.Sprint(time.Now().Unix())
	user, err := getLoginInfoByToken("admin:" + password.Md5Encode(password.Encode("secret")+ts) + ":" + ts)
	if err != nil || user.ID != 1 {
		t.Fatalf("valid existing API token rejected: %v", err)
	}
}

func TestSessionRevalidatesPasswordDisabledAndPrivileges(t *testing.T) {
	setupHTTPAuthDB(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", contextIterceptor(controllers.Login))
	mux.HandleFunc("/api/user/info", contextIterceptor(func(ctx *context.Context, w http.ResponseWriter, r *http.Request) {
		response.NewSuccessResponse(ctx.IsAdmin).FPrint(w)
	}))
	handler := session.Instance.LoadAndSave(mux)
	call := func(path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	login := func(old *http.Cookie) *http.Cookie {
		w := call("/api/login", `{"account":"admin","password":"secret"}`, old)
		cookies := w.Result().Cookies()
		if len(cookies) == 0 || !strings.Contains(w.Body.String(), `"errorNo":0`) {
			t.Fatalf("login failed: %s", w.Body)
		}
		return cookies[0]
	}
	cookie := login(nil)
	renewed := login(cookie)
	if renewed.Value == cookie.Value {
		t.Fatal("successful login did not rotate session token")
	}
	cookie = renewed
	if _, err := db.Instance.Exec("UPDATE user SET is_admin=0 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if w := call("/api/user/info", "{}", cookie); !strings.Contains(w.Body.String(), `"data":false`) {
		t.Fatalf("stale admin privileges: %s", w.Body)
	}
	if _, err := db.Instance.Exec("UPDATE user SET disabled=1 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	assertLoginRequired(t, call("/api/user/info", "{}", cookie))
	if _, err := db.Instance.Exec("UPDATE user SET disabled=0 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	cookie = login(nil)
	if _, err := db.Instance.Exec("UPDATE user SET password=? WHERE id=1", password.Encode("changed")); err != nil {
		t.Fatal(err)
	}
	assertLoginRequired(t, call("/api/user/info", "{}", cookie))
}

func assertLoginRequired(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	var res response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || res.ErrorNo != response.NeedLogin {
		t.Fatalf("expected expired session: %s", w.Body)
	}
}

func TestBrowserRequestOriginAndMethods(t *testing.T) {
	setupHTTPAuthDB(t)
	for _, tc := range []struct {
		origin, site string
		want         bool
	}{
		{"", "", true}, {"https://mail.example.com", "same-origin", true}, {"https://evil.example", "cross-site", false},
		{"null", "", false}, {"https://mail.example.com.evil.test", "", false}, {"https://evil.example.com", "same-site", false},
	} {
		r := httptest.NewRequest("POST", "http://mail.example.com/api/email/send", nil)
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("Sec-Fetch-Site", tc.site)
		if allowBrowserRequest(r) != tc.want {
			t.Fatalf("origin=%q site=%q", tc.origin, tc.site)
		}
	}
	handler := contextIterceptor(func(*context.Context, http.ResponseWriter, *http.Request) { t.Fatal("mutation was called") })
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest("GET", "/api/email/send", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET mutation status=%d", w.Code)
	}
}
