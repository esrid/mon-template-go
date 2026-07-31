package identity

import (
	"bytes"
	"testing"
)

func TestTokenIsRandomAndHashable(t *testing.T) {
	first, firstHash, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two generated tokens are equal")
	}
	if !bytes.Equal(firstHash, tokenHash(first)) {
		t.Fatal("stored hash does not match token")
	}
}
