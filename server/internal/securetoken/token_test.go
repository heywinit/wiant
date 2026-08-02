package securetoken

import (
	"bytes"
	"testing"
)

func TestNewReturnsUniqueTokensAndMatchingDigests(t *testing.T) {
	first, firstDigest, err := New()
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("expected unique tokens")
	}
	if !bytes.Equal(firstDigest, Digest(first)) {
		t.Fatal("returned digest does not match token")
	}
}
