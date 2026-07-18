package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrNotFound = gorm.ErrRecordNotFound

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

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
	if err := s.db.WithContext(ctx).Preload("AllowedProviders").Preload("AllowedModels.Provider").Where("api_key_hash = ?", apiKeyHash).First(&user).Error; err != nil {
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
// this is temporary
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

func (s *Store) GetBaseURLForModel(ctx context.Context, modelName string, authorization string) (string, string, error) {
	authorization = strings.TrimPrefix(authorization, "Bearer ")
	var models []Model
	result := s.db.WithContext(ctx).Preload("Provider").Find(&models)

	if result.Error != nil {
		return "", "", result.Error
	}

	if len(models) == 0 {
		return "", "", fmt.Errorf("no models found in the database")
	}

	for _, model := range models {
		if model.Model == modelName {
			return model.Provider.BaseURL, model.Model, nil
		}
		parts := strings.Split(model.Model, "/")
		if len(parts) == 2 && parts[1] == modelName {
			return model.Provider.BaseURL, model.Model, nil
		}
	}
	return "", "", fmt.Errorf("model %s not found", modelName)
}
