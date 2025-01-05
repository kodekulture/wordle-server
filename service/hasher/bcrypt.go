package hasher

import (
	"github.com/lordvidex/errs/v2"
	"golang.org/x/crypto/bcrypt"
)

type Bcrypt struct{}

func (b *Bcrypt) Hash(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (b *Bcrypt) Compare(hashed, original string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(original))
	if err != nil {
		return errs.WrapCode(err, errs.Unauthenticated, "passwords do not match")
	}
	return nil
}
