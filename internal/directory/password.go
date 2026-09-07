package directory

import (
	"errors"

	"github.com/woodleighschool/goodies/auth/authn"
)

// ErrWeakPassword rejects local passwords shorter than 12 bytes.
var ErrWeakPassword = errors.New("password must be at least 12 characters")

func hashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", ErrWeakPassword
	}
	return authn.HashPassword(password)
}
