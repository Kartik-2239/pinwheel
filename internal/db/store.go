package db

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ErrNotFound is returned when a queried row does not exist.
var ErrNotFound = gorm.ErrRecordNotFound

// Store provides the data access layer over the proxy database.
type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

// CreateUser validates and inserts a new user, applying the default rate
// limit when none is set.
func (s *Store) CreateUser(ctx context.Context, user *User) error {
	if user.Name == "" || user.APIKeyHash == "" {
		return fmt.Errorf("name and api_key_hash cannot be empty")
	}
	if user.RateLimit5hr == nil {
		defaultVal := DefaultRateLimit5hr
		user.RateLimit5hr = &defaultVal
	}
	return s.db.WithContext(ctx).Create(user).Error
}

func (s *Store) GetUserByHash(ctx context.Context, apiKeyHash string) (*User, error) {
	var user User
	if err := s.db.WithContext(ctx).Where("api_key_hash = ?", apiKeyHash).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	var users []User
	err := s.db.WithContext(ctx).Find(&users).Error
	return users, err
}

// UpdateUserUsage increments the user's token counter and stamps last_used_at.
func (s *Store) UpdateUserUsage(ctx context.Context, id uint, tokens int64) error {
	res := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(map[string]any{
		"tokens_used":  gorm.Expr("tokens_used + ?", tokens),
		"last_used_at": time.Now().UTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}


