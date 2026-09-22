package smtp_server

import (
	"bytes"
	stdcontext "context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/hooks"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/mileusna/spf"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"xorm.io/xorm"
)

func testInit(t *testing.T) {
	t.Helper()
	oldRoot, oldConfig, oldInit := config.ROOT_PATH, config.Instance, config.IsInit
	oldDB, oldHooks, oldResolver := db.Instance, hooks.HookList, net.DefaultResolver
	oldSPFServer := spf.DNSServer
	t.Cleanup(func() {
		config.ROOT_PATH, config.Instance, config.IsInit = oldRoot, oldConfig, oldInit
		db.Instance, hooks.HookList, net.DefaultResolver = oldDB, oldHooks, oldResolver
		spf.DNSServer = oldSPFServer
	})
	config.ROOT_PATH = t.TempDir() + "/"
	keyPath := filepath.Join(config.ROOT_PATH, "config/dkim/dkim.priv")
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0600); err != nil {
		t.Fatal(err)
	}
	config.Instance = &config.Config{
		Domain: "fixture.example", Domains: []string{"fixture.example", "linuxuser.site", "jinnrry.com", "jiangwei.one", "qq.com"},
		DbType: config.DBTypeSQLite, DbDSN: filepath.Join(config.ROOT_PATH, "config/fixture.db"),
		DkimPrivateKeyPath: keyPath, CatchAllAccount: "admin", AcceptSubdomains: true, IsInit: true,
	}
	config.IsInit = true
	if err = config.WriteConfig(config.Instance); err != nil {
		t.Fatal(err)
	}
	engine, err := xorm.NewEngine("sqlite", config.Instance.DbDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	engine.SetMaxOpenConns(1)
	if err = engine.Sync2(&models.User{}, &models.Email{}, &models.UserEmail{}, &models.Rule{}); err != nil {
		t.Fatal(err)
	}
	if _, err = engine.Insert(&models.User{Account: "admin", Name: "Fixture admin", IsAdmin: 1}); err != nil {
		t.Fatal(err)
	}
	db.Instance = engine
	// These historical MIME fixtures exercise receiving, not notification or
	// forwarding integrations. Never start plugins, listeners or real sessions.
	hooks.HookList = nil
	// Exercise the real SPF/DKIM failure paths without querying public DNS.
	// This is test-scoped dependency replacement, not a production auth bypass.
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(stdcontext.Context, string, string) (net.Conn, error) {
		return nil, errors.New("DNS disabled for offline SMTP fixture")
	}}
	// The SPF library has its own DNS client. Its deliberately invalid address
	// fails before opening a socket, even if a future fixture sets MAIL FROM.
	spf.DNSServer = "127.0.0.1" // missing port: deterministic resolver failure
}

func receiveFixture(t *testing.T, session *Session, message string) {
	t.Helper()
	before, err := db.Instance.Count(&models.Email{})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Data(bytes.NewReader([]byte(message))); err != nil {
		t.Fatalf("receive fixture: %v", err)
	}
	count, err := db.Instance.Count(&models.Email{})
	if err != nil || count != before+1 {
		t.Fatalf("fixture not persisted exactly once: before=%d count=%d err=%v", before, count, err)
	}
}

func TestOfflineFixtureSPFReturnsFailureWithoutNetwork(t *testing.T) {
	testInit(t)
	if result := spf.CheckHost(net.ParseIP("203.0.113.1"), "example.com", "sender@example.com", "mx.example.com"); result != spf.TempError {
		t.Fatalf("offline SPF resolver result = %v, want TempError", result)
	}
}

func TestGmail(t *testing.T) {
	testInit(t)
	emailData := `Received: by mail-dl1-f46.google.com with SMTP id a92af1059eb24-132c338a537so5798485c88.0
        for ; Sun, 24 May 2026 08:18:19 -0700 (PDT)
ARC-Seal: i=1; a=rsa-sha256; t=1779635898; cv=none;
        d=google.com; s=arc-20240605;
        b=T2MoTE0s0bFADLxUOXXGL5f6JyO2V39Pcae2DJ21wpFO3D5b8YXzGP5Qq8BO5y3ya+
         a2Huxddo3xEJikr9FnvBztxWuwyekSpqM+/NH8Do/ZTEconsziakmma08oLO+Uw5+fxO
         OmV3oo9N2MtWm7nM1EGkUZ3H1EpEAE7nfQ//gXrEGog+u3dKLns5aS9LHl7fjmHUK4ll
         +mWj3AcbiBLjOzfV2iSCq8kLD8QgN8kDugXzsMLoXxchzblMMw/G6Z1gMTDtRyFPdEOp
         eKn70lz4dRJyGIdsPbS2N4hSBLNHMrQ3YIUDJ3UO2/HfTG8uenKAQznt5jZJixKa9hiT
         +W+g==
ARC-Message-Signature: i=1; a=rsa-sha256; c=relaxed/relaxed; d=google.com; s=arc-20240605;
        h=to:subject:message-id:date:from:mime-version:dkim-signature;
        bh=krMNvdL0wIWIbww3tkM/xlqwm94sXWplcXqGF4i93iM=;
        fh=6iav8edVevT+IuZxBKkoHjEHmIBsC8UMPcjZALicuAw=;
        b=WqsmQI22ntkTMVXOTnajcX141N5/B/Xh6R0N5e+LmT8WzQAPNOPK7r/o7ALBmPEA/t
         3y5z8EsUhxNUnSaQmDoRGz4AhC/PvDlxVhE4SS0rEVGxJFmCwpPFb+QhgDEyRh0WUqcO
         /4hQ7rFqWN7oJb5YBuJOX241VHW8jFrQAuzgdLPQXSYPz7Rc4gL0pDIj/1GfWVKwEqbT
         U/oB7u99oaagwwN1u5U2kphaTxCmaa1syxqpNFEcirDGMwf5WaNpOuFj0zgIip7SMEJY
         AtD0WxQaO2fN7gbiDO62ZoqkgFjKJ3G3psreDP0oS1zXRn8xHeBFnoqUXOVhZCIDlVmb
         FT2A==;
        darn=linuxuser.site
ARC-Authentication-Results: i=1; mx.google.com; arc=none
DKIM-Signature: v=1; a=rsa-sha256; c=relaxed/relaxed;
        d=gmail.com; s=20251104; t=1779635898; x=1780240698; darn=linuxuser.site;
        h=to:subject:message-id:date:from:mime-version:from:to:cc:subject
         :date:message-id:reply-to;
        bh=krMNvdL0wIWIbww3tkM/xlqwm94sXWplcXqGF4i93iM=;
        b=c/8+TeXpadbyFUf68OwKCCq5Oot/asRNC38R0/hUp1gGm6rJvV9ly7ffClrFogdbH+
         SJRUFMNBDuu5/Tzyuk/4SlWK5sEWC29y0i00pVmOFoet55hm58mLgdmrtpIdCQ6L773R
         7akIMo3Q6D074PUiK/pxOEGUNp2A7pmZb2q9ZQUeZ/dfblgJh6Q/u+8N1o+nDwabnUPF
         H/nRaXoA3YdTt0s5tXr0+wDLW4MpXOudo2Of2ELHOzmXA93WJDE1dWNX/xuaF5W9s4nk
         dVt4A97Zui4t3cPdZEjvrG6yY+t5Fu5gKmVVpVTtvRJITWIqhzxiaxWtQtlbKLK+LD8U
         3nfQ==
X-Google-DKIM-Signature: v=1; a=rsa-sha256; c=relaxed/relaxed;
        d=1e100.net; s=20251104; t=1779635898; x=1780240698;
        h=to:subject:message-id:date:from:mime-version:x-gm-gg
         :x-gm-message-state:from:to:cc:subject:date:message-id:reply-to;
        bh=krMNvdL0wIWIbww3tkM/xlqwm94sXWplcXqGF4i93iM=;
        b=LLWfy51jPWQLEt/9Vs8TzwMojaDTfB7cUcYc5IdQAn7SScU+N4kqeTLb8aK9fmTeX/
         G8ZytpEbKbXAyHtNxh0ezs5ro03UljCTEzKZGEcB2Qy0Idjd2vY9saRXbFU/7h7/BdNm
         Tztf1K5Wvf+ZmQyYGVLb6yJbGv/RTql6ZsLcym+aWv6JEkVtRVMwJ/IHm951eQrur8KE
         tsiOafkuHRZzkSDSq8hqw61/v/APnjuTVB66UhC8hbAErav9tlIAScqbzSwSimxB/Aj0
         2OTvo6hHRCsc+I2/6gtkTDZKam/DWVs/hA9hURGRvQORos7Yk3r5snAn5mHB1ybJCrpw
         7New==
X-Gm-Message-State: AOJu0Yy7VeDNPEFVk1g6RqPM+hyzjrIVL2deO813Iq4W+3/H6+06AsZG
     vQbBZjD+YbasztrgwM8FGpqRDiF7NlauyDQyhPzL9NZ7TKa0y0Q5gWah+FKMBcXFOI7r11JyTb/
     5Dh6yLwCQ2BWrluabz0fPRL/XUXQNmwFGS9du1f4=
X-Gm-Gg: Acq92OFQjCPQ1C3jnZQayhjIMDz/3uQ0AVjPNBUEdpuEvw5pkJXnD7s6j8PEKgf9jUA
     gbc+koh+f+m2686BdRuMX58AIQ689Q4WpcdEq4lRpbKZpoasjniA5u1l/lBI1/+i0olLrkHO0LF
     EF5QEemC93jbaR8hB7EMPoZ4zmIIe0tOqaChCMthhtKp/df2kiLc+BIFVJR04EKqEi48tsdOrrM
     6qNGo3oufUSAFfhJKUNUF88UhOzWAyfc6t9lEdby+Q+LwOopdenMnV837OlsTIpQMKHJayw6aoW
     049SxRoJ/Q==
X-Received: by 2002:a05:7022:6b95:b0:123:3c24:b15 with SMTP id
 a92af1059eb24-136340ce619mr5527125c88.19.1779635898132; Sun, 24 May 2026
 08:18:18 -0700 (PDT)
MIME-Version: 1.0
From: Ming Deer 
Date: Sun, 24 May 2026 23:18:07 +0800
X-Gm-Features: AVHnY4KsQv0XWokXwRtSxwial70kRdR-hEcbdeCsV7DylqP4mkAA0SzorBeWNMA
Message-ID: 
Subject: =?UTF-8?B?5Li76aKY5rWL6K+VMg==?=
To: xfox@linuxuser.site 
Content-Type: multipart/alternative; boundary="00000000000089ad93065291c5f4"

--00000000000089ad93065291c5f4
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: base64

5q2j5paH5rWL6K+VMg0K
--00000000000089ad93065291c5f4
Content-Type: text/html; charset="UTF-8"
Content-Transfer-Encoding: base64

PGRpdiBkaXI9Imx0ciI+PGRpdj7mraPmlofmtYvor5UyPC9kaXY+PC9kaXY+DQo=
--00000000000089ad93065291c5f4--
`

	s := Session{
		RemoteAddress: net.TCPAddrFromAddrPort(netip.AddrPortFrom(netip.AddrFrom4([4]byte{}), 25)),
		Ctx:           &context.Context{},
		To:            []string{"xfox@linuxuser.site"},
	}

	receiveFixture(t, &s, emailData)

}

func TestPmailEmail(t *testing.T) {
	testInit(t)
	emailData := `DKIM-Signature: a=rsa-sha256; bh=x7Rh+N2y2K9exccEAyKCTAGDgYKfnLZpMWc25ug5Ny4=;
 c=simple/simple; d=domain.com;
 h=Content-Type:Mime-Version:Subject:To:From:Date; s=default; t=1693831868;
 v=1;
 b=1PZEupYvSMtGyYx42b4G65YbdnRj4y2QFo9kS7GXiTVhUM5EYzJhZzknwRMN5RL5aFY26W4E
 DmzJ85XvPPvrDtnU/B4jkc5xthE+KEsb1Go8HcL8WQqwvsE9brepeA0t0RiPnA/x7dbTo3u72SG
 WqtviWbJH5lPFc9PkSbEPFtc=
Content-Type: multipart/mixed;
 boundary=3c13260efb7bd8bad8315c21215489fe283f36cdf82813674f6e11215f6c
Mime-Version: 1.0
Subject: =?utf-8?q?=E6=8F=92=E4=BB=B6=E6=B5=8B=E8=AF=95?=
To: =?utf-8?q?=E5=90=8D?= <ok@jinnrry.com>
From: =?utf-8?q?=E5=8F=91=E9=80=81=E4=BA=BA?= <j@jinnrry.com>
Date: Mon, 04 Sep 2023 20:51:08 +0800

--3c13260efb7bd8bad8315c21215489fe283f36cdf82813674f6e11215f6c
Content-Type: multipart/alternative;
 boundary=9ebf2f3c4f97c51dd9a285ae28a54d2d0d84aa6d0ad28b76547e2096bb66

--9ebf2f3c4f97c51dd9a285ae28a54d2d0d84aa6d0ad28b76547e2096bb66
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/plain

=E8=BF=99=E6=98=AFText
--9ebf2f3c4f97c51dd9a285ae28a54d2d0d84aa6d0ad28b76547e2096bb66
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/html

<div>=E8=BF=99=E6=98=AFHtml</div>
--9ebf2f3c4f97c51dd9a285ae28a54d2d0d84aa6d0ad28b76547e2096bb66--

--3c13260efb7bd8bad8315c21215489fe283f36cdf82813674f6e11215f6c--
`
	s := Session{
		RemoteAddress: net.TCPAddrFromAddrPort(netip.AddrPortFrom(netip.AddrFrom4([4]byte{}), 25)),
		Ctx: &context.Context{
			UserID:      0,
			UserName:    "",
			UserAccount: "",
		},
		To: []string{"ok@jinnrry.com"},
	}

	receiveFixture(t, &s, emailData)

}

func TestRuleForward(t *testing.T) {
	testInit(t)

	forwardEmail := `DKIM-Signature: a=rsa-sha256; bh=bpOshF+iimuqAQijVxqkH6gPpWf8A+Ih30/tMjgEgS0=;
 c=simple/simple; d=jinnrry.com;
 h=Content-Type:Mime-Version:Subject:To:From:Date; s=default; t=1693992640;
 v=1;
 b=XiOgYL9iGrkuYzXBAf7DSO0sRbFr6aPOE4VikmselNKEF1UTjMPdiqpeHyx/i6BOQlJWWZEC
 PzceHTDFIStcZE6a5Sc1nh8Fis+gRkrheBO/zK/P5P/euK+0Fj5+0T82keNTSCgo1ZtEIubaNR0
 JvkwJ2ZC9g8xV6Yiq+ZhRriT8lZ6zeI55PPEFJIzFgZ7xDshDgx5E7J1xRXQqcEMV1rgVq04d3c
 6wjU+LLtghmgtUToRp3ASn6DhVO+Bbc4QkmcQ/StQH3681+1GVMHvQSBhSSymSRA71SikE2u3a1
 JnvbOP9fThP7h+6oFEIRuF7MwDb3JWY5BXiFFKCkecdFg==
Content-Type: multipart/mixed;
 boundary=8e9d5abb6bdac11b8d7d6e13280af1a87d12b904a59368d6e852b0a4ce3e
Mime-Version: 1.0
Subject: forward
To: <t@jiangwei.one>
From: "i" <i@jinnrry.com>
Date: Wed, 06 Sep 2023 17:30:40 +0800

--8e9d5abb6bdac11b8d7d6e13280af1a87d12b904a59368d6e852b0a4ce3e
Content-Type: multipart/alternative;
 boundary=a62ae91c159ea22e8196d57d344626eb00d1ddfa9c5064a39b01588aa992

--a62ae91c159ea22e8196d57d344626eb00d1ddfa9c5064a39b01588aa992
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/plain

hello pls Forward the email.
--a62ae91c159ea22e8196d57d344626eb00d1ddfa9c5064a39b01588aa992
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/html

<p>hello pls Forward the email.</p>
--a62ae91c159ea22e8196d57d344626eb00d1ddfa9c5064a39b01588aa992--

--8e9d5abb6bdac11b8d7d6e13280af1a87d12b904a59368d6e852b0a4ce3e--`

	readEmail := `DKIM-Signature: a=rsa-sha256; bh=JcCDj6edb1bAwRbcFZ63plFZOeB5AdGWLE/PQ2FQ1Tc=;
 c=simple/simple; d=jinnrry.com;
 h=Content-Type:Mime-Version:Subject:To:From:Date; s=default; t=1693992600;
 v=1;
 b=rwlqSkDFKYH42pA1jsajemaw+4YdeLHPeqV4mLQrRdihgma1VSvXl5CEOur/KuwQuUarr2cu
 SntWrHE6+RnDaQcPEHbkgoMjEJw5+VPwkIvE6VSlMIB7jg93mGzvN2yjheWTePZ+cVPjOaIrgir
 wiT24hkrTHp+ONT8XoS0sDuY+ieyBZp/GCv/YvgE4t0JEkNozMAVWotrXxaICDzZoWP3NNmKLqg
 6He6zwWAl51r3W5R5weGBi6A/FqlHgHZGroXnNi+wolDuN6pQiVAJ7MZ6hboPCbCCRrBQDTdor5
 wEI2+MwlJ/d2f17wxoGmluCewbeYttuVcpUOVwACJKw3g==
Content-Type: multipart/mixed;
 boundary=9e33a130a8a976102a93e296d6408d228e151f7841ca9ee0d777234fd6f3
Mime-Version: 1.0
Subject: read
To: <t@jiangwei.one>
From: "i" <i@jinnrry.com>
Date: Wed, 06 Sep 2023 17:30:00 +0800

--9e33a130a8a976102a93e296d6408d228e151f7841ca9ee0d777234fd6f3
Content-Type: multipart/alternative;
 boundary=54a95f3429f3cdb342383db10293780bed341f8dc20d2f876eb0853e3884

--54a95f3429f3cdb342383db10293780bed341f8dc20d2f876eb0853e3884
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/plain

12 aRead 1sadf
--54a95f3429f3cdb342383db10293780bed341f8dc20d2f876eb0853e3884
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/html

<p>12 aRead 1sadf</p>
--54a95f3429f3cdb342383db10293780bed341f8dc20d2f876eb0853e3884--

--9e33a130a8a976102a93e296d6408d228e151f7841ca9ee0d777234fd6f3--`

	moveEmail := `DKIM-Signature: a=rsa-sha256; bh=YQfG/wlHGhky6FNmpIwgDYDOc/uyivdBv+9S02Z04xY=;
 c=simple/simple; d=jinnrry.com;
 h=Content-Type:Mime-Version:Subject:To:From:Date; s=default; t=1693992542;
 v=1;
 b=IhxswOCq8I7CmCas1EMp+n8loR7illqlF0IJC6eN1+OLjI/E5BPzpP4HWkyqaAkd0Vn9i+Bn
 MVb5kNHZ2S7qt0rqAAc6Atc0i9WpLEI3Cng+VDn+difcMZlJSAkhLLn2sUsS4Fzqqo3Cbw62qSO
 TgnWRmlj9aM+5xfGcl/76WOvQQpahJbGg6Go51kFMeHVom/VeGKIgFBCeMe37T/LS03c3pAV8gA
 i6Zy3GYE57W/qU3oCzaGeS3n5zom/i74H4VipiVIMX/OBNYhdHWrP8vyjvzLFpJlXp6RvzcRl0P
 ytyiCZfE8G7fAFntp20LW70Y5Xgqqczk1jR578UDczVoA==
Content-Type: multipart/mixed;
 boundary=c84d60b253aa6caee345c73e717ad59b1975448bbdfad7a23ac4d76e022d
Mime-Version: 1.0
Subject: Move
To: <t@jiangwei.one>
From: "i" <i@jinnrry.com>
Date: Wed, 06 Sep 2023 17:29:02 +0800

--c84d60b253aa6caee345c73e717ad59b1975448bbdfad7a23ac4d76e022d
Content-Type: multipart/alternative;
 boundary=a69985ebcf3c1c44d6e69e5a29c1044743cd9e44d4bc9bb6886f83a73966

--a69985ebcf3c1c44d6e69e5a29c1044743cd9e44d4bc9bb6886f83a73966
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/plain

MOVE move Move
--a69985ebcf3c1c44d6e69e5a29c1044743cd9e44d4bc9bb6886f83a73966
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/html

<p>MOVE move Move</p>
--a69985ebcf3c1c44d6e69e5a29c1044743cd9e44d4bc9bb6886f83a73966--

--c84d60b253aa6caee345c73e717ad59b1975448bbdfad7a23ac4d76e022d--`

	deleteEmail := `DKIM-Signature: a=rsa-sha256; bh=dNtHGqd1NbRj0WSwrJmPsqAcAy3h/4kZK2HFQ0Asld8=;
 c=simple/simple; d=jinnrry.com;
 h=Content-Type:Mime-Version:Subject:To:From:Date; s=default; t=1693992495;
 v=1;
 b=QllU8lqGdoOMaGYp8d13oWytb7+RebqKjq4y8Rs/kOeQxoE8dSEVliK3eBiXidsNTdDtkTqf
 eiwjyRBK92NVCYprdJqLbu9qZ39BC2lk3NXttTSJ1+1ZZ/bGtIW5JIYn2pToED0MqVVkxGFUtl+
 qFmc4mWo5a4Mbij7xaAB3uJtHpBDt7q4Ovr2hiMetQv7YrhZvCt/xrH8Q9YzZ6xzFUL5ekW40eH
 oWElU1GyVBHWCKh31aweyhA+1XLPYojjREQYd4svRqTbSFSsBqFwFIUGdnyJh2WgmF8eucmttAw
 oRhgzyZkHL1jAskKFBpO10SDReyk50Cvc+0kSLj+QcUpg==
Content-Type: multipart/mixed;
 boundary=bdfa9bf94e22e218105281e06bd59bd6df3ce70e71367bf49fbe73301af3
Mime-Version: 1.0
Subject: test
To: <t@jiangwei.one>
From: "i" <i@jinnrry.com>
Date: Wed, 06 Sep 2023 17:28:15 +0800

--bdfa9bf94e22e218105281e06bd59bd6df3ce70e71367bf49fbe73301af3
Content-Type: multipart/alternative;
 boundary=7352524eaae801790245f6bf095460fd1f4e01f5748b4dba48635bf59b04

--7352524eaae801790245f6bf095460fd1f4e01f5748b4dba48635bf59b04
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/plain

Delete
--7352524eaae801790245f6bf095460fd1f4e01f5748b4dba48635bf59b04
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/html

<p>Delete</p>
--7352524eaae801790245f6bf095460fd1f4e01f5748b4dba48635bf59b04--

--bdfa9bf94e22e218105281e06bd59bd6df3ce70e71367bf49fbe73301af3--`

	s := Session{
		RemoteAddress: net.TCPAddrFromAddrPort(netip.AddrPortFrom(netip.AddrFrom4([4]byte{}), 25)),
		Ctx:           &context.Context{},
	}

	receiveFixture(t, &s, deleteEmail)
	receiveFixture(t, &s, readEmail)
	receiveFixture(t, &s, forwardEmail)
	receiveFixture(t, &s, moveEmail)
}

func TestRuleRead(t *testing.T) {
	testInit(t)

	readEmail := `DKIM-Signature: a=rsa-sha256; bh=JcCDj6edb1bAwRbcFZ63plFZOeB5AdGWLE/PQ2FQ1Tc=;
 c=simple/simple; d=jinnrry.com;
 h=Content-Type:Mime-Version:Subject:To:From:Date; s=default; t=1693992600;
 v=1;
 b=rwlqSkDFKYH42pA1jsajemaw+4YdeLHPeqV4mLQrRdihgma1VSvXl5CEOur/KuwQuUarr2cu
 SntWrHE6+RnDaQcPEHbkgoMjEJw5+VPwkIvE6VSlMIB7jg93mGzvN2yjheWTePZ+cVPjOaIrgir
 wiT24hkrTHp+ONT8XoS0sDuY+ieyBZp/GCv/YvgE4t0JEkNozMAVWotrXxaICDzZoWP3NNmKLqg
 6He6zwWAl51r3W5R5weGBi6A/FqlHgHZGroXnNi+wolDuN6pQiVAJ7MZ6hboPCbCCRrBQDTdor5
 wEI2+MwlJ/d2f17wxoGmluCewbeYttuVcpUOVwACJKw3g==
Content-Type: multipart/mixed;
 boundary=9e33a130a8a976102a93e296d6408d228e151f7841ca9ee0d777234fd6f3
Mime-Version: 1.0
Subject: read
To: <t@jiangwei.one>
From: "i" <i@jinnrry.com>
Date: Wed, 06 Sep 2023 17:30:00 +0800

--9e33a130a8a976102a93e296d6408d228e151f7841ca9ee0d777234fd6f3
Content-Type: multipart/alternative;
 boundary=54a95f3429f3cdb342383db10293780bed341f8dc20d2f876eb0853e3884

--54a95f3429f3cdb342383db10293780bed341f8dc20d2f876eb0853e3884
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/plain

12 aRead 1sadf
--54a95f3429f3cdb342383db10293780bed341f8dc20d2f876eb0853e3884
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/html

<p>12 aRead 1sadf</p>
--54a95f3429f3cdb342383db10293780bed341f8dc20d2f876eb0853e3884--

--9e33a130a8a976102a93e296d6408d228e151f7841ca9ee0d777234fd6f3--`

	s := Session{
		RemoteAddress: net.TCPAddrFromAddrPort(netip.AddrPortFrom(netip.AddrFrom4([4]byte{}), 25)),
		Ctx:           &context.Context{},
	}

	receiveFixture(t, &s, readEmail)

}

func TestRuleDelete(t *testing.T) {
	testInit(t)

	deleteEmail := `DKIM-Signature: a=rsa-sha256; bh=dNtHGqd1NbRj0WSwrJmPsqAcAy3h/4kZK2HFQ0Asld8=;
 c=simple/simple; d=jinnrry.com;
 h=Content-Type:Mime-Version:Subject:To:From:Date; s=default; t=1693992495;
 v=1;
 b=QllU8lqGdoOMaGYp8d13oWytb7+RebqKjq4y8Rs/kOeQxoE8dSEVliK3eBiXidsNTdDtkTqf
 eiwjyRBK92NVCYprdJqLbu9qZ39BC2lk3NXttTSJ1+1ZZ/bGtIW5JIYn2pToED0MqVVkxGFUtl+
 qFmc4mWo5a4Mbij7xaAB3uJtHpBDt7q4Ovr2hiMetQv7YrhZvCt/xrH8Q9YzZ6xzFUL5ekW40eH
 oWElU1GyVBHWCKh31aweyhA+1XLPYojjREQYd4svRqTbSFSsBqFwFIUGdnyJh2WgmF8eucmttAw
 oRhgzyZkHL1jAskKFBpO10SDReyk50Cvc+0kSLj+QcUpg==
Content-Type: multipart/mixed;
 boundary=bdfa9bf94e22e218105281e06bd59bd6df3ce70e71367bf49fbe73301af3
Mime-Version: 1.0
Subject: test
To: <t@jiangwei.one>
From: "i" <i@jinnrry.com>
Date: Wed, 06 Sep 2023 17:28:15 +0800

--bdfa9bf94e22e218105281e06bd59bd6df3ce70e71367bf49fbe73301af3
Content-Type: multipart/alternative;
 boundary=7352524eaae801790245f6bf095460fd1f4e01f5748b4dba48635bf59b04

--7352524eaae801790245f6bf095460fd1f4e01f5748b4dba48635bf59b04
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/plain

Delete
--7352524eaae801790245f6bf095460fd1f4e01f5748b4dba48635bf59b04
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/html

<p>Delete</p>
--7352524eaae801790245f6bf095460fd1f4e01f5748b4dba48635bf59b04--

--bdfa9bf94e22e218105281e06bd59bd6df3ce70e71367bf49fbe73301af3--`

	s := Session{
		RemoteAddress: net.TCPAddrFromAddrPort(netip.AddrPortFrom(netip.AddrFrom4([4]byte{}), 25)),
		Ctx:           &context.Context{},
	}

	receiveFixture(t, &s, deleteEmail)

}

func TestNullCC(t *testing.T) {
	testInit(t)

	emailData := `Date: Mon, 29 Jan 2024 16:54:30 +0800
Return-Path: 1231@111.com
From: =?utf-8?B?b2VhdHY=?= 1231@111.com
To: =?utf-8?B?ODQ2ODAzOTY=?= 123213@qq.com
Cc:
Bcc:
Reply-To: <>
Subject: =?utf-8?B?6L+Z5piv5LiA5bCB5p2l6IeqUmVsYXhEcmFtYeeahOmCruS7tg==?=
Message-ID: <cf43cc780b72dad392d4f90dfced88a8@1231@111.com>
X-Priority: 3
X-Mailer: Mailer (https://github.com/txthinking/Mailer)
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="6edc2ef285d93010a080caccc858c67b"

--6edc2ef285d93010a080caccc858c67b
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: base64

PGRpdiBzdHlsZT0ibWluLWhlaWdodDo1NTBweDsgcGFkZGluZzogMTAwcHggNTVweCAyMDBweDsi
Pui/meaYr+S4gOWwgeadpeiHqlJlbGF4RHJhbWHnmoTmoKHpqozpgq7ku7Ys55So5LqO5qCh6aqM
6YKu5Lu26YWN572u5piv5ZCm5q2j5bi4ITwvZGl2Pg==

--6edc2ef285d93010a080caccc858c67b
Content-Type: text/html; charset="UTF-8"
Content-Transfer-Encoding: base64

PGRpdiBzdHlsZT0ibWluLWhlaWdodDo1NTBweDsgcGFkZGluZzogMTAwcHggNTVweCAyMDBweDsi
Pui/meaYr+S4gOWwgeadpeiHqlJlbGF4RHJhbWHnmoTmoKHpqozpgq7ku7Ys55So5LqO5qCh6aqM
6YKu5Lu26YWN572u5piv5ZCm5q2j5bi4ITwvZGl2Pg==

--6edc2ef285d93010a080caccc858c67b--`
	s := Session{
		RemoteAddress: net.TCPAddrFromAddrPort(netip.AddrPortFrom(netip.AddrFrom4([4]byte{}), 25)),
		Ctx:           &context.Context{},
	}

	receiveFixture(t, &s, emailData)
}

func TestRuleMove(t *testing.T) {
	testInit(t)

	moveEmail := `DKIM-Signature: a=rsa-sha256; bh=YQfG/wlHGhky6FNmpIwgDYDOc/uyivdBv+9S02Z04xY=;
 c=simple/simple; d=jinnrry.com;
 h=Content-Type:Mime-Version:Subject:To:From:Date; s=default; t=1693992542;
 v=1;
 b=IhxswOCq8I7CmCas1EMp+n8loR7illqlF0IJC6eN1+OLjI/E5BPzpP4HWkyqaAkd0Vn9i+Bn
 MVb5kNHZ2S7qt0rqAAc6Atc0i9WpLEI3Cng+VDn+difcMZlJSAkhLLn2sUsS4Fzqqo3Cbw62qSO
 TgnWRmlj9aM+5xfGcl/76WOvQQpahJbGg6Go51kFMeHVom/VeGKIgFBCeMe37T/LS03c3pAV8gA
 i6Zy3GYE57W/qU3oCzaGeS3n5zom/i74H4VipiVIMX/OBNYhdHWrP8vyjvzLFpJlXp6RvzcRl0P
 ytyiCZfE8G7fAFntp20LW70Y5Xgqqczk1jR578UDczVoA==
Content-Type: multipart/mixed;
 boundary=c84d60b253aa6caee345c73e717ad59b1975448bbdfad7a23ac4d76e022d
Mime-Version: 1.0
Subject: Move
To: <t@jiangwei.one>
From: "i" <i@jinnrry.com>
Date: Wed, 06 Sep 2023 17:29:02 +0800

--c84d60b253aa6caee345c73e717ad59b1975448bbdfad7a23ac4d76e022d
Content-Type: multipart/alternative;
 boundary=a69985ebcf3c1c44d6e69e5a29c1044743cd9e44d4bc9bb6886f83a73966

--a69985ebcf3c1c44d6e69e5a29c1044743cd9e44d4bc9bb6886f83a73966
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/plain

MOVE move Move
--a69985ebcf3c1c44d6e69e5a29c1044743cd9e44d4bc9bb6886f83a73966
Content-Transfer-Encoding: quoted-printable
Content-Disposition: inline
Content-Type: text/html

<p>MOVE move Move</p>
--a69985ebcf3c1c44d6e69e5a29c1044743cd9e44d4bc9bb6886f83a73966--

--c84d60b253aa6caee345c73e717ad59b1975448bbdfad7a23ac4d76e022d--`

	s := Session{
		RemoteAddress: net.TCPAddrFromAddrPort(netip.AddrPortFrom(netip.AddrFrom4([4]byte{}), 25)),
		Ctx:           &context.Context{},
	}

	receiveFixture(t, &s, moveEmail)
}

func TestQAEmailForward(t *testing.T) {
	testInit(t)
	data := `Mime-Version: 1.0
X-QQ-MIME: TCMime 1.0 by Tencent
X-Mailer: QQMail 2.x
X-QQ-Mailer: QQMail 2.x
Message-ID: tencent_D82739970C66D2BFBA23F4A3@qq.com
Subject: =?UTF-8?B?5rWL6K+V5Y+R6YCB?=
Date: Wed, 10 Apr 2024 11:11:12 +0800 (GMT+08:00)
From: =?UTF-8?B?YWRtaW5AamlubnJyeS5jb20=?=<admin@jinnrry.com>
To: =?UTF-8?B??=<test@jinnrry.com>
Content-Type: multipart/alternative; 
        boundary="----=_Part_174_107154538.1712718674768"

------=_Part_174_107154538.1712718674768
Content-Type: text/plain; charset=us-ascii
Content-Transfer-Encoding: base64


------=_Part_174_107154538.1712718674768
Content-Type: text/html; charset=UTF-8
Content-Transfer-Encoding: base64

PGRpdj7ov5nph4zmmK/lhoXlrrk8L2Rpdj48ZGl2PjwhLS1lbXB0eXNpZ24tLT48L2Rpdj4=
------=_Part_174_107154538.1712718674768--`

	s := Session{
		RemoteAddress: net.TCPAddrFromAddrPort(netip.AddrPortFrom(netip.AddrFrom4([4]byte{}), 25)),
		Ctx:           &context.Context{},
	}

	receiveFixture(t, &s, data)
}
