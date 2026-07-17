package db

import "time"

// DefaultRateLimit5hr is applied to new users that don't specify a limit.
const DefaultRateLimit5hr = 1500

// User is an API key holder. The plaintext key is never stored — only its
// SHA-256 hash and the last 4 characters for identification.
// Nil limit fields mean "unlimited".
type User struct {
	ID          uint       `gorm:"primaryKey"`
	Name        string     `gorm:"uniqueIndex;not null"`
	APIKeyHash  string     `gorm:"uniqueIndex;not null"`
	Last4Digits string     `gorm:"column:last_4_digits;not null"`
	LastUsedAt  *time.Time // nil until first use
	CreatedAt   time.Time  `gorm:"not null"`

	RateLimit5hr *int   // requests per 5-hour window
	MaxTokens    *int64 // total token budget
	TokensUsed   int64  `gorm:"not null;default:0"`
}


