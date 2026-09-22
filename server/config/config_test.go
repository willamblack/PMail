package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeConfigRejectsMalformedAndNull(t *testing.T) {
	for _, data := range []string{"null", "{", "[]", `{"isInit":true, "domain":42}`} {
		if _, err := decodeConfig([]byte(data)); err == nil {
			t.Fatalf("accepted invalid config %s", data)
		}
	}
}

func TestInitInvalidConfigDoesNotReplaceRunningState(t *testing.T) {
	oldRoot, oldInstance, oldInit, oldArgs := ROOT_PATH, Instance, IsInit, os.Args
	ROOT_PATH = t.TempDir() + "/"
	Instance = &Config{Domain: "existing.example", IsInit: true}
	IsInit = true
	os.Args = []string{"pmail"}
	t.Cleanup(func() { ROOT_PATH, Instance, IsInit, os.Args = oldRoot, oldInstance, oldInit, oldArgs })
	if err := os.MkdirAll(filepath.Join(ROOT_PATH, "config"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ROOT_PATH, "config/config.json"), []byte(`{"domain":"partial.example", "isInit":`), 0600); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Error("invalid config did not fail closed")
		}
		if Instance.Domain != "existing.example" || !IsInit {
			t.Error("invalid config replaced running state")
		}
	}()
	Init()
}

func TestFixPathResolvesRelativeDKIMKey(t *testing.T) {
	oldRoot := ROOT_PATH
	ROOT_PATH = "/example/runtime/"
	t.Cleanup(func() { ROOT_PATH = oldRoot })
	cfg := &Config{DkimPrivateKeyPath: "config/dkim/dkim.priv"}
	cfg.fixPath()
	if cfg.DkimPrivateKeyPath != "/example/runtime/config/dkim/dkim.priv" {
		t.Fatalf("wrong DKIM path: %s", cfg.DkimPrivateKeyPath)
	}
}

func TestReadConfigPreservesLegacySubdomainOptIn(t *testing.T) {
	oldRoot := ROOT_PATH
	ROOT_PATH = t.TempDir() + "/"
	t.Cleanup(func() { ROOT_PATH = oldRoot })
	if err := WriteConfig(&Config{Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AcceptSubdomains {
		t.Fatal("reading existing config silently enabled subdomains")
	}
}

func TestACMEAccountKeyCreationAndInvalidExistingKey(t *testing.T) {
	oldRoot := ROOT_PATH
	ROOT_PATH = t.TempDir() + "/"
	t.Cleanup(func() { ROOT_PATH = oldRoot })
	key, fresh, err := ReadPrivateKey()
	if err != nil || key == nil || !fresh {
		t.Fatalf("new key: fresh=%v err=%v", fresh, err)
	}
	path := filepath.Join(ROOT_PATH, "config/ssl/account_private.pem")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("private key permissions %o", info.Mode().Perm())
	}
	loaded, fresh, err := ReadPrivateKey()
	if err != nil || fresh || !key.Equal(loaded) {
		t.Fatalf("key reload: fresh=%v err=%v", fresh, err)
	}
	if err = os.WriteFile(path, []byte("invalid PEM"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = ReadPrivateKey(); err == nil {
		t.Fatal("invalid existing account key was silently replaced")
	}
}
