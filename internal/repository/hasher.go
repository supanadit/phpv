package repository

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"strings"
)

// NewHasher returns a hash.Hash for the given checksum type.
// Supported types: "sha256".
func NewHasher(checksumType string) (hash.Hash, error) {
	switch strings.ToLower(checksumType) {
	case "sha256":
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("unsupported checksum type: %s", checksumType)
	}
}
