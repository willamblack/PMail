package setup

import (
	"path/filepath"
	"testing"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	_ "modernc.org/sqlite"
	"xorm.io/xorm"
)

func TestSetAdminPasswordAssignsCatchAllAccount(t *testing.T) {
	oldRoot := config.ROOT_PATH
	oldConfig := config.Instance
	oldDB := db.Instance
	config.ROOT_PATH = filepath.ToSlash(t.TempDir()) + "/"
	config.Instance = &config.Config{}

	engine, err := xorm.NewEngine("sqlite", filepath.Join(t.TempDir(), "setup.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err = engine.Sync2(&models.User{}); err != nil {
		t.Fatal(err)
	}
	db.Instance = engine
	t.Cleanup(func() {
		_ = engine.Close()
		db.Instance = oldDB
		config.Instance = oldConfig
		config.ROOT_PATH = oldRoot
	})

	if err = SetAdminPassword(&context.Context{}, "owner", "secret"); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.ReadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CatchAllAccount != "owner" || config.Instance.CatchAllAccount != "owner" {
		t.Fatalf("catch-all account = file:%q runtime:%q, want owner", cfg.CatchAllAccount, config.Instance.CatchAllAccount)
	}
}
