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
	CreatedAt   time.Time  `gorm:"autoCreateTime"`

	RateLimit5hr *int   // requests per 5-hour window
	MaxTokens    *int64 // total token budget
	TokensUsed   int64  `gorm:"not null;default:0"`

	AllowedProviders []Provider `gorm:"many2many:user_providers;"`
	AllowedModels    []Model    `gorm:"many2many:user_models;"`
}

type Usage struct {
	ID         uint      `gorm:"primaryKey"`
	UserID     uint      `gorm:"index:idx_user_time,priority:1;not null"`
	CreatedAt  time.Time `gorm:"index:idx_user_time,priority:2;not null"`
	Provider   string    `gorm:"not null"`
	Model      string    `gorm:"not null"`
	TokensIn   int64     `gorm:"not null;default:0"`
	TokensOut  int64     `gorm:"not null;default:0"`
	CostMicros int64     `gorm:"not null;default:0"` // 1 millionth of a dollar

	User *User `gorm:"constraint:OnDelete:CASCADE"`
}

type Provider struct {
	ID      uint   `gorm:"primaryKey"`
	Name    string `gorm:"unique;not null"`
	BaseURL string `gorm:"not null"`
	EnvKey  string `gorm:"not null"`
}

type Model struct {
	ID    uint   `gorm:"primaryKey"`
	Model string `gorm:"not null"`

	ProviderID uint     `gorm:"not null"`
	Provider   Provider `gorm:"constraint:OnDelete:CASCADE"`
}
