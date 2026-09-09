// Package auth handles password hashing and opaque session tokens.
package auth

import "golang.org/x/crypto/bcrypt"

// bcryptCost is the work factor for password hashing.
const bcryptCost = 12

// HashPassword returns a bcrypt hash of the plaintext password.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword reports whether plain matches the given bcrypt hash.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// dummyHash is compared against on the "user not found" login path so that
// response timing does not reveal whether an email is registered.
var dummyHash []byte

func init() {
	dummyHash, _ = bcrypt.GenerateFromPassword([]byte("commitin/timing-equalizer"), bcryptCost)
}

// CheckDummy performs a throwaway bcrypt comparison to equalize timing.
func CheckDummy(plain string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(plain))
}
