package http_server

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/controllers"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/i18n"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/session"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/httputil"
	"github.com/Jinnrry/pmail/utils/id"
	"github.com/Jinnrry/pmail/utils/password"
	olog "log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

//go:embed dist/*
var local embed.FS

var httpsServer *http.Server

type nullWrite struct {
}

func (w *nullWrite) Write(p []byte) (int, error) {
	return len(p), nil
}

func HttpsStart() {

	mux := http.NewServeMux()

	router(mux)

	// go http server会打一堆没用的日志，写一个空的日志处理器，屏蔽掉日志输出
	nullLog := olog.New(&nullWrite{}, "", olog.Ldate)

	HttpsPort := 443
	if config.Instance.HttpsPort > 0 {
		HttpsPort = config.Instance.HttpsPort
	}

	if config.Instance.HttpsEnabled != 2 {
		log.Infof("Https Server Start On Port :%d", HttpsPort)
		httpsServer = &http.Server{
			Addr:         fmt.Sprintf(":%d", HttpsPort),
			Handler:      normalizeLeadingSlashes(session.Instance.LoadAndSave(mux)),
			ReadTimeout:  time.Second * 90,
			WriteTimeout: time.Second * 90,
			ErrorLog:     nullLog,
		}
		err := httpsServer.ListenAndServeTLS(config.Instance.SSLPublicKeyPath, config.Instance.SSLPrivateKeyPath)
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				// 正常关闭（重启或停机）
				log.Infof("Https Server closed normally on port :%d", HttpsPort)
			} else {
				// 异常错误仍然打印并退出
				log.Errorf("Https Server error on port :%d, %+v", HttpsPort, err)
				panic(err)
			}
		}
	}
}

func HttpsStop() {
	if httpsServer != nil {
		httpsServer.Close()
	}
}

// 新增：分类处理关闭错误

// 注入context
func contextIterceptor(h controllers.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if !allowBrowserRequest(r) {
			http.Error(w, "Cross-origin request denied", http.StatusForbidden)
			return
		}
		if requiresPost(r.URL.Path) && r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, httputil.MaxJSONBodySize)
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}

		ctx := &context.Context{}
		ctx.Context = r.Context()
		ctx.SetValue(context.LogID, id.GenLogID())
		lang := r.Header.Get("Lang")
		if lang == "" {
			lang = "en"
		}
		ctx.Lang = lang

		if config.IsInit {
			user := cast.ToString(session.Instance.Get(ctx, "user"))
			var userInfo models.User
			if user != "" {
				_ = json.Unmarshal([]byte(user), &userInfo)
			}
			if userInfo.ID > 0 {
				// Revalidate privileges and credentials on every request. A cached
				// admin/disabled flag must not outlive an account or password change.
				var current models.User
				found, err := db.Instance.ID(userInfo.ID).Where("disabled=0").Get(&current)
				stamp := session.Instance.GetString(ctx, "credential_version")
				if err != nil {
					response.NewErrorResponse(response.ServerError, "Unable to verify session", "").FPrint(w)
					return
				}
				if !found || stamp == "" || subtle.ConstantTimeCompare([]byte(stamp), []byte(session.CredentialVersion(current.Password))) != 1 {
					_ = session.Instance.Destroy(ctx)
					userInfo = models.User{}
				} else {
					ctx.UserID = current.ID
					ctx.UserName = current.Name
					ctx.UserAccount = current.Account
					ctx.IsAdmin = current.IsAdmin == 1
				}
			}

			if userInfo.ID == 0 {
				token := r.Header.Get("Token")
				if token != "" {
					user, err := getLoginInfoByToken(token)
					if user.ID > 0 && err == nil {
						ctx.UserID = user.ID
						ctx.UserName = user.Name
						ctx.UserAccount = user.Account
						ctx.IsAdmin = user.IsAdmin == 1
					} else {
						response.NewErrorResponse(response.NeedLogin, "Invalid or expired token", "").FPrint(w)
						return
					}
				}
			}

			if ctx.UserID == 0 {
				if r.URL.Path != "/api/ping" && r.URL.Path != "/api/login" {
					response.NewErrorResponse(response.NeedLogin, i18n.GetText(ctx.Lang, "login_exp"), "").FPrint(w)
					return
				}
			}
		} else if r.URL.Path != "/api/setup" {
			response.NewErrorResponse(response.NeedSetup, "", "").FPrint(w)
			return
		}
		h(ctx, w, r)
	}
}

/*
Token 格式：  {account}:md5(md5(md5(password+"pmail")+"pmail2023")+{Unix timestamp}):{Unix timestamp}
比如：         admin::
*/
func getLoginInfoByToken(token string) (models.User, error) {
	ret := models.User{}
	data := strings.Split(token, ":")
	if len(data) != 3 {
		return ret, errors.New("token format error")
	}
	account := data[0]
	encodePwd := data[1]
	requestTimeStamp, err := strconv.ParseInt(data[2], 10, 64)
	now := time.Now().Unix()
	if err != nil || requestTimeStamp > now || requestTimeStamp <= now-5 {
		return ret, errors.New("token expired")
	}
	var user models.User
	found, err := db.Instance.Table("user").Where("account =? and disabled=0", account).Get(&user)
	if err != nil || !found || user.ID == 0 {
		return ret, errors.New("account or password error")
	}
	tokenPwd := password.Md5Encode(fmt.Sprintf("%s%d", user.Password, requestTimeStamp))
	if subtle.ConstantTimeCompare([]byte(tokenPwd), []byte(encodePwd)) != 1 {
		return ret, errors.New("account or password error")
	}

	return user, nil
}

func requiresPost(path string) bool {
	switch path {
	case "/api/login", "/api/logout", "/api/setup", "/api/group/add", "/api/group/del",
		"/api/email/del", "/api/email/read", "/api/email/detail", "/api/email/move", "/api/email/send",
		"/api/settings/modify_password", "/api/rule/add", "/api/rule/update", "/api/rule/del",
		"/api/user/create", "/api/user/edit":
		return true
	}
	return false
}

func allowBrowserRequest(r *http.Request) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // CLI/API clients do not send Origin; still require authentication.
	}
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host) ||
		(config.Instance.WebDomain != "" && strings.EqualFold(u.Host, config.Instance.WebDomain))
}
