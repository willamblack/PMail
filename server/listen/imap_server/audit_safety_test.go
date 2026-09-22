package imap_server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/models"
	pcontext "github.com/Jinnrry/pmail/utils/context"
	"github.com/emersion/go-imap/v2"
)

func configureIMAPTests(root string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, DNSNames: []string{"localhost"}}
	cert, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPath, certPath := filepath.Join(root, "key.pem"), filepath.Join(root, "cert.pem")
	if err = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0600); err != nil {
		panic(err)
	}
	if err = os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}), 0600); err != nil {
		panic(err)
	}
	config.ROOT_PATH = root + string(os.PathSeparator)
	config.Instance = &config.Config{Domain: "example.test", DbType: config.DBTypeSQLite, DbDSN: filepath.Join(root, "imap.db"), SSLPrivateKeyPath: keyPath, SSLPublicKeyPath: certPath}
}

func TestAuditDeletedFlagsRequireExpunge(t *testing.T) {
	const user = 8181
	var ids []int
	for _, status := range []int8{0, 0, 5} {
		email := models.Email{Subject: "audit-delete"}
		if _, err := db.Instance.Insert(&email); err != nil {
			t.Fatal(err)
		}
		ue := models.UserEmail{UserID: user, EmailID: email.Id, Status: status}
		if _, err := db.Instance.Insert(&ue); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, ue.ID)
	}
	s := &serverSession{ctx: &pcontext.Context{UserID: user}, currentMailbox: "INBOX"}
	set := imap.UIDSetNum(imap.UID(ids[0]), imap.UID(ids[1]))
	if err := s.Store(nil, set, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Flags: []imap.Flag{imap.FlagDeleted}, Silent: true}, nil); err != nil {
		t.Fatal(err)
	}
	assertPresent := func(id int, want bool) {
		t.Helper()
		got, err := db.Instance.ID(id).Exist(&models.UserEmail{})
		if err != nil || got != want {
			t.Fatalf("id %d present=%v want=%v err=%v", id, got, want, err)
		}
	}
	assertPresent(ids[0], true)
	if err := s.Store(nil, imap.UIDSetNum(imap.UID(ids[0])), &imap.StoreFlags{Op: imap.StoreFlagsDel, Flags: []imap.Flag{imap.FlagDeleted}, Silent: true}, nil); err != nil {
		t.Fatal(err)
	}
	// A malicious/unmarked UID and another mailbox must never be removed.
	s.deleteUidList = append(s.deleteUidList, ids[2])
	huge := imap.UIDSet{{Start: 1, Stop: imap.UID(^uint32(0))}}
	if err := s.Expunge(nil, &huge); err != nil {
		t.Fatal(err)
	}
	assertPresent(ids[0], true)
	assertPresent(ids[1], false)
	assertPresent(ids[2], true)
	s.readOnly = true
	if err := s.Store(nil, set, &imap.StoreFlags{}, nil); err == nil {
		t.Fatal("read-only STORE succeeded")
	}
	if err := s.Expunge(nil, nil); err == nil {
		t.Fatal("read-only EXPUNGE succeeded")
	}
}

func TestAuditAppendNeverClaimsSuccess(t *testing.T) {
	s := &serverSession{ctx: &pcontext.Context{}}
	if _, err := s.Append("INBOX", nil, nil); err == nil {
		t.Fatal("unimplemented APPEND claimed success")
	}
}

func TestAuditRangesAndMoveDoNotTouchOtherCopies(t *testing.T) {
	const user = 8282
	email := models.Email{Subject: "range-copy"}
	if _, err := db.Instance.Insert(&email); err != nil {
		t.Fatal(err)
	}
	var uids []imap.UID
	for i := 0; i < 3; i++ {
		ue := models.UserEmail{UserID: user, EmailID: email.Id}
		if _, err := db.Instance.Insert(&ue); err != nil {
			t.Fatal(err)
		}
		uids = append(uids, imap.UID(ue.ID))
	}
	s := &serverSession{ctx: &pcontext.Context{UserID: user}, currentMailbox: "INBOX"}
	rows := s.messages(imap.UIDSetNum(uids[2]))
	if len(rows) != 1 || rows[0].SerialNumber != 3 {
		t.Fatalf("UID subset renumbered: %+v", rows)
	}
	rows = s.messages(imap.SeqSet{{Start: 3, Stop: 1}})
	if len(rows) != 3 {
		t.Fatalf("reversed range got %d", len(rows))
	}
	rows = s.messages(imap.SeqSet{{Start: 0, Stop: 0}})
	if len(rows) != 1 || rows[0].SerialNumber != 3 {
		t.Fatal("star did not select last message")
	}
	copied, err := s.Copy(imap.UIDSetNum(uids[0], uids[2]), "Junk")
	if err != nil || len(copied.SourceUIDs) != 2 {
		t.Fatalf("disjoint COPY lost range: %+v %v", copied, err)
	}
	if err = s.Move(nil, imap.UIDSetNum(uids[1]), "Junk"); err != nil {
		t.Fatal(err)
	}
	remaining := s.messages(imap.SeqSet{{Start: 1, Stop: 0}})
	if len(remaining) != 2 {
		t.Fatalf("MOVE affected other copies: %d", len(remaining))
	}
}
