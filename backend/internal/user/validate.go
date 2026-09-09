package user

import (
	"fmt"
	"regexp"
	"strings"
)

// Field constraints. Username charset is lowercase-only by design so that
// uniqueness is plain (no case-folding needed).
const (
	UsernameMinLen = 3
	UsernameMaxLen = 30
	PasswordMinLen = 8
	PasswordMaxLen = 72 // bcrypt only hashes the first 72 bytes
)

var (
	usernameRe = regexp.MustCompile(`^[a-z0-9_-]+$`)
	// Deliberately loose: a single "@" with non-empty local and domain parts
	// that contains a dot. Real validation is sending mail, which v1 does not do.
	emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

// ValidateUsername checks length and charset. It does not check uniqueness.
func ValidateUsername(username string) error {
	u := strings.ToLower(strings.TrimSpace(username))
	if len(u) < UsernameMinLen || len(u) > UsernameMaxLen {
		return fmt.Errorf("username must be %d-%d characters", UsernameMinLen, UsernameMaxLen)
	}
	if !usernameRe.MatchString(u) {
		return fmt.Errorf("username may only contain lowercase letters, digits, '-' and '_'")
	}
	return nil
}

// ValidateEmail does a shape check only.
func ValidateEmail(email string) error {
	e := strings.ToLower(strings.TrimSpace(email))
	if len(e) > 254 || !emailRe.MatchString(e) {
		return fmt.Errorf("invalid email address")
	}
	return nil
}

// ValidatePassword checks length bounds.
func ValidatePassword(password string) error {
	if len(password) < PasswordMinLen {
		return fmt.Errorf("password must be at least %d characters", PasswordMinLen)
	}
	if len(password) > PasswordMaxLen {
		return fmt.Errorf("password must be at most %d characters", PasswordMaxLen)
	}
	return nil
}
