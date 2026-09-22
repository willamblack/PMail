package match

import (
	"fmt"
	"github.com/Jinnrry/pmail/dto/parsemail"
	"github.com/Jinnrry/pmail/utils/context"
	"github.com/dlclark/regexp2"
	log "github.com/sirupsen/logrus"
	"time"
)

type RegexMatch struct {
	Rule  string
	Field string
}

func NewRegexMatch(field, rule string) *RegexMatch {
	return &RegexMatch{
		Rule:  rule,
		Field: field,
	}
}

func (r *RegexMatch) Match(ctx *context.Context, email *parsemail.Email) bool {
	content := getFieldContent(r.Field, email)
	re, err := compileRegex(r.Rule)
	if err != nil {
		log.WithContext(ctx).Warn("Invalid mail rule regular expression")
		return false
	}
	match, err := re.MatchString(content)

	if err != nil {
		// regexp2 errors may embed the complete email body. Do not log it.
		log.WithContext(ctx).Warn("Mail rule regular expression failed or timed out")
	}

	return err == nil && match
}

func compileRegex(pattern string) (*regexp2.Regexp, error) {
	if len(pattern) > 4096 {
		return nil, fmt.Errorf("regular expression exceeds 4096 bytes")
	}
	re, err := regexp2.Compile(pattern, 0)
	if err == nil {
		// Email bodies are controlled by remote senders. Backtracking must not
		// monopolize an SMTP worker even if an administrator supplied the rule.
		re.MatchTimeout = 100 * time.Millisecond
	}
	return re, err
}

func ValidateRegex(pattern string) error {
	_, err := compileRegex(pattern)
	return err
}
