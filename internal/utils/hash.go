package utils

import (
	"crypto/sha256"
	"fmt"
)

func HashString(s string) string {
	// Use a simple hash function for demonstration purposes
	hash := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", hash[:])
}
