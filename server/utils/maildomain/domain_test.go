package maildomain

import "testing"

func TestMatchRoot(t *testing.T) {
	roots := []string{"117799.xyz", "Example.COM", "mail.example.com."}
	tests := []struct {
		name             string
		domain           string
		acceptSubdomains bool
		wantRoot         string
		want             bool
	}{
		{name: "root", domain: "117799.xyz", acceptSubdomains: true, wantRoot: "117799.xyz", want: true},
		{name: "one label", domain: "a.117799.xyz", acceptSubdomains: true, wantRoot: "117799.xyz", want: true},
		{name: "many labels", domain: "a.b.c.117799.xyz", acceptSubdomains: true, wantRoot: "117799.xyz", want: true},
		{name: "case and trailing dot", domain: "A.117799.XYZ.", acceptSubdomains: true, wantRoot: "117799.xyz", want: true},
		{name: "longest root", domain: "x.mail.example.com", acceptSubdomains: true, wantRoot: "mail.example.com", want: true},
		{name: "subdomains disabled", domain: "a.117799.xyz", acceptSubdomains: false, want: false},
		{name: "missing label boundary", domain: "evil117799.xyz", acceptSubdomains: true, want: false},
		{name: "suffix after root", domain: "117799.xyz.evil.com", acceptSubdomains: true, want: false},
		{name: "unconfigured", domain: "example.org", acceptSubdomains: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, ok := MatchRoot(tt.domain, roots, tt.acceptSubdomains)
			if ok != tt.want || root != tt.wantRoot {
				t.Fatalf("MatchRoot(%q) = (%q, %t), want (%q, %t)", tt.domain, root, ok, tt.wantRoot, tt.want)
			}
		})
	}
}

func TestNormalizeIDN(t *testing.T) {
	got, err := Normalize("例子.测试.")
	if err != nil {
		t.Fatal(err)
	}
	if got != "xn--fsqu00a.xn--0zwm56d" {
		t.Fatalf("Normalize() = %q", got)
	}
}

func TestSplitAddressUsesFinalAt(t *testing.T) {
	local, domain, err := SplitAddress(`"a@b"@A.117799.XYZ.`)
	if err != nil {
		t.Fatal(err)
	}
	if local != `"a@b"` || domain != "a.117799.xyz" {
		t.Fatalf("SplitAddress() = (%q, %q)", local, domain)
	}
}

func TestEqualAddressNormalizesDomainOnly(t *testing.T) {
	if !EqualAddress("Admin@A.117799.XYZ.", "Admin@a.117799.xyz") {
		t.Fatal("same address with normalized domain should compare equal")
	}
	if EqualAddress("Admin@117799.xyz", "admin@117799.xyz") {
		t.Fatal("local-part comparison must remain case-sensitive")
	}
}
