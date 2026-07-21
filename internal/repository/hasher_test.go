package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestNewHasher_SHA256(t *testing.T) {
	h, err := NewHasher("sha256")
	if err != nil {
		t.Fatalf("NewHasher(sha256) returned error: %v", err)
	}

	h.Write([]byte("hello"))
	got := hex.EncodeToString(h.Sum(nil))
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Fatalf("sha256(hello) = %q, want %q", got, want)
	}
}

func TestNewHasher_CaseInsensitive(t *testing.T) {
	h, err := NewHasher("SHA256")
	if err != nil {
		t.Fatalf("NewHasher(SHA256) returned error: %v", err)
	}

	h.Write([]byte("hello"))
	got := hex.EncodeToString(h.Sum(nil))
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Fatalf("SHA256(hello) = %q, want %q", got, want)
	}
}

func TestNewHasher_UnsupportedType(t *testing.T) {
	h, err := NewHasher("md5")
	if err == nil {
		t.Fatal("NewHasher(md5) expected error, got nil")
	}
	if h != nil {
		t.Fatalf("NewHasher(md5) returned non-nil hasher: %v", h)
	}
}

func TestNewHasher_EmptyType(t *testing.T) {
	h, err := NewHasher("")
	if err == nil {
		t.Fatal("NewHasher(\"\") expected error, got nil")
	}
	if h != nil {
		t.Fatalf("NewHasher(\"\") returned non-nil hasher: %v", h)
	}
}

func TestNewHasher_MatchesCryptoSHA256(t *testing.T) {
	input := []byte("the quick brown fox jumps over the lazy dog")

	got, err := NewHasher("sha256")
	if err != nil {
		t.Fatalf("NewHasher(sha256) returned error: %v", err)
	}
	got.Write(input)

	want := sha256.Sum256(input)
	if !equalHash(got.Sum(nil), want[:]) {
		t.Fatalf("NewHasher result does not match crypto/sha256")
	}
}

func equalHash(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
