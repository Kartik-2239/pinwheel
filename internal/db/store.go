package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Kartik-2239/pinwheel/internal/utils"
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

func (s *Store) GetModelFromName(ctx context.Context, modelName string, key string) ([]Model, error) {
	key = strings.TrimPrefix(key, "Bearer ")
	if key == "" {
		return nil, fmt.Errorf("api key cannot be empty")
	}

	var user User
	if err := s.db.WithContext(ctx).
		Preload("AllowedModels.Provider").
		Where("api_key_hash = ?", utils.HashString(key)).
		First(&user).Error; err != nil {
		return nil, err
	}
	models := []Model{}
	for _, model := range user.AllowedModels {
		qualifiedName := model.Provider.Name + "/" + model.Model
		parts := strings.Split(model.Model, "/")
		if strings.EqualFold(model.Model, modelName) || strings.EqualFold(qualifiedName, modelName) || len(parts) == 2 && strings.EqualFold(parts[1], modelName) {
			// return []Model{model}, nil
			models = append(models, model)
		}
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("model %q is not allowed", modelName)
	}
	return models, nil
}

func (s *Store) CreateUsage(ctx context.Context, apiKey string, modelName string, provider string, tokensIn int64, tokensOut int64, costMicros *int64) error {
	apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	if apiKey == "" {
		return fmt.Errorf("api key cannot be empty")
	}

	var user User
	if err := s.db.WithContext(ctx).Where("api_key_hash = ?", utils.HashString(apiKey)).First(&user).Error; err != nil {
		return err
	}

	// var model Model
	// if err := s.db.WithContext(ctx).Preload("Provider").Where("model = ?", modelName).First(&model).Error; err == nil && model.Provider.Name != "" {
	// 	provider = model.Provider.Name
	// } else if parts := strings.SplitN(modelName, "/", 2); len(parts) == 2 {
	// 	provider = parts[0]
	// }

	cost := int64(0)
	if costMicros != nil {
		cost = *costMicros
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&Usage{
			UserID:     user.ID,
			CreatedAt:  time.Now().UTC(),
			Provider:   provider,
			Model:      modelName,
			TokensIn:   tokensIn,
			TokensOut:  tokensOut,
			CostMicros: cost,
		}).Error; err != nil {
			return err
		}

		return tx.Model(&User{}).Where("id = ?", user.ID).Updates(map[string]any{
			"tokens_used":  gorm.Expr("tokens_used + ?", tokensIn+tokensOut),
			"last_used_at": time.Now().UTC(),
		}).Error
	})
}

func (s *Store) GetTotalCost(ctx context.Context, userID uint) (int64, error) {
	var totalCost int64
	err := s.db.WithContext(ctx).Model(&Usage{}).Where("user_id = ?", userID).Select("SUM(cost_micros)").Scan(&totalCost).Error
	if err != nil {
		return 0, err
	}
	return totalCost, nil
}
