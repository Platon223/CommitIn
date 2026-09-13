package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Platon223/commitin/cli/internal/config"
)

func TestRequireLoginWithToken(t *testing.T) {
	var out bytes.Buffer
	if err := requireLogin(&out, &config.Config{Token: "tok"}); err != nil {
		t.Fatalf("requireLogin with a token returned an error: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("requireLogin with a token printed something: %q", out.String())
	}
}

func TestRequireLoginWithoutToken(t *testing.T) {
	var out bytes.Buffer
	err := requireLogin(&out, &config.Config{})
	if err == nil {
		t.Fatal("requireLogin without a token returned nil error")
	}
	if !strings.Contains(out.String(), "cmtin signup") || !strings.Contains(out.String(), "cmtin login") {
		t.Fatalf("requireLogin message doesn't point to signup/login: %q", out.String())
	}
}
