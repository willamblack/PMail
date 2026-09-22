package match

import (
	"fmt"
	"github.com/Jinnrry/pmail/dto/parsemail"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/dlclark/regexp2"
	"strings"
	"testing"
	"time"
)

func TestRegexMatch_Match(t *testing.T) {
	re := regexp2.MustCompile("^(?!.*abc\\.com).*", 0)
	match, err := re.MatchString("aa@abc.com")
	fmt.Println(match, err)
}

func TestUntrustedRegexFailsSafely(t *testing.T) {
	for _, pattern := range []string{"[", strings.Repeat("a", 4097)} {
		if ValidateRegex(pattern) == nil {
			t.Fatalf("accepted invalid pattern %q", pattern)
		}
		if NewRegexMatch("Text", pattern).Match(&context.Context{}, &parsemail.Email{Text: []byte("mail")}) {
			t.Fatal("invalid rule matched")
		}
	}
	started := time.Now()
	if NewRegexMatch("Text", `^(a+)+$`).Match(&context.Context{}, &parsemail.Email{Text: []byte(strings.Repeat("a", 10000) + "!")}) {
		t.Fatal("timed-out rule matched")
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("regex deadline was not honored")
	}
	if getFieldContent("From", &parsemail.Email{}) != "" || getFieldContent("Sender", nil) != "" {
		t.Fatal("missing identities must yield empty fields")
	}
}
