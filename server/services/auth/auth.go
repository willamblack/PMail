package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"sync"
)

// HasAuth 检查当前用户是否有某个邮件的auth
func HasAuth(ctx *context.Context, email *models.Email) bool {
	if ctx == nil || ctx.UserID <= 0 || email == nil || email.Id <= 0 {
		return false
	}
	if ctx.IsAdmin {
		return true
	}
	var ue []models.UserEmail
	err := db.Instance.Table(&models.UserEmail{}).Where("email_id = ? and user_id = ?", email.Id, ctx.UserID).Find(&ue)
	if err != nil {
		log.Errorf("Error while checking user: %v", err)
		return false
	}

	return len(ue) != 0
}

var dkimMu sync.Mutex

// DkimGen derives DNS data from the configured private key. A missing public
// file must never silently rotate an existing key and invalidate published DNS.
func DkimGen(privatePath string) (string, error) {
	dkimMu.Lock()
	defer dkimMu.Unlock()
	if privatePath == "" {
		return "", fmt.Errorf("DKIM private key path is empty")
	}
	publicPath := filepath.Join(filepath.Dir(privatePath), "dkim.public")
	if filepath.Clean(privatePath) == publicPath {
		return "", fmt.Errorf("DKIM private key must not use the public key output path")
	}
	keyBytes, err := os.ReadFile(privatePath)
	var key *rsa.PrivateKey
	if os.IsNotExist(err) {
		key, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return "", err
		}
		der, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(privatePath), 0700); err != nil {
			return "", err
		}
		f, err := os.OpenFile(privatePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return "", err
		}
		writeErr := pem.Encode(f, &pem.Block{Type: "PRIVATE KEY", Bytes: der})
		closeErr := f.Close()
		if writeErr != nil {
			return "", writeErr
		}
		if closeErr != nil {
			return "", closeErr
		}
	} else if err != nil {
		return "", err
	} else {
		block, _ := pem.Decode(keyBytes)
		if block == nil {
			return "", fmt.Errorf("invalid DKIM private key PEM")
		}
		if block.Type == "RSA PRIVATE KEY" {
			key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		} else {
			var parsed any
			parsed, err = x509.ParsePKCS8PrivateKey(block.Bytes)
			key, _ = parsed.(*rsa.PrivateKey)
		}
		if err != nil || key == nil {
			return "", fmt.Errorf("invalid RSA DKIM private key")
		}
	}
	if err := key.Validate(); err != nil {
		return "", err
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", err
	}
	publicKey := "v=DKIM1; k=rsa; p=" + base64.StdEncoding.EncodeToString(pubBytes)
	if err := os.WriteFile(publicPath, []byte(publicKey), 0644); err != nil {
		return "", err
	}
	return publicKey, nil
}
