package identity

import (
	"errors"
	"testing"
)

func TestBcryptHasher(t *testing.T) {
	hasher := NewBcryptHasher(4)
	hash, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if err := hasher.Compare(hash, "correct horse battery staple"); err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if !errors.Is(hasher.Compare(hash, "wrong password"), ErrInvalidCredentials) {
		t.Fatal("wrong password was accepted")
	}
	if hasher.NeedsRehash(hash) {
		t.Fatal("fresh hash unexpectedly needs rehash")
	}
}

func TestBcryptRejectsLongPassword(t *testing.T) {
	password := make([]byte, 73)
	for i := range password {
		password[i] = 'a'
	}
	_, err := NewBcryptHasher(4).Hash(string(password))
	if !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("Hash() error = %v", err)
	}
}
