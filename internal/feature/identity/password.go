package identity

import "golang.org/x/crypto/bcrypt"

const DefaultBcryptCost = 12

type BcryptHasher struct{ Cost int }

func NewBcryptHasher(cost int) BcryptHasher {
	if cost == 0 {
		cost = DefaultBcryptCost
	}
	return BcryptHasher{Cost: cost}
}

func (h BcryptHasher) Hash(password string) (string, error) {
	if len(password) < 8 {
		return "", ErrInvalidPassword
	}
	if len([]byte(password)) > 72 {
		return "", ErrPasswordTooLong
	}
	value, err := bcrypt.GenerateFromPassword([]byte(password), h.Cost)
	return string(value), err
}

func (h BcryptHasher) Compare(hash, password string) error {
	if len([]byte(password)) > 72 {
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

func (h BcryptHasher) NeedsRehash(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	return err != nil || cost != h.Cost
}
