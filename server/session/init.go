package session

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/db"
	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"net/http"

	"time"
)

var Instance *scs.SessionManager

func Init() {
	Instance = scs.New()
	Instance.Lifetime = 7 * 24 * time.Hour
	Instance.Cookie.HttpOnly = true
	Instance.Cookie.SameSite = http.SameSiteLaxMode
	Instance.Cookie.Secure = config.Instance.HttpsEnabled != 2
	// 使用db存储session数据，目前为了架构简单，
	// 暂不引入redis存储，如果日后性能存在瓶颈，可以将session迁移到redis

	switch config.Instance.DbType {
	case config.DBTypeMySQL:
		Instance.Store = mysqlstore.New(db.Instance.DB().DB)
	case config.DBTypeSQLite:
		Instance.Store = sqlite3store.New(db.Instance.DB().DB)
	case config.DBTypePostgres:
		Instance.Store = postgresstore.New(db.Instance.DB().DB)
	default:
		panic("Unsupported database type: " + config.Instance.DbType)
	}

}

// CredentialVersion lets password changes invalidate existing sessions without
// storing the password/hash itself in the serialized login information.
func CredentialVersion(encodedPassword string) string {
	sum := sha256.Sum256([]byte(encodedPassword))
	return hex.EncodeToString(sum[:])
}
