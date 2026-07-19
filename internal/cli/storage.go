package cli

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/Kartik-2239/openai-proxy/internal/db"
	"github.com/Kartik-2239/openai-proxy/internal/utils"
)

func createAPIKey(database *gorm.DB, name string, choices []modelChoice, costLimit *int64, expiration *time.Time) (string, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return "", err
	}

	providers, models, err := saveAllowedModels(database, choices)
	if err != nil {
		return "", err
	}

	user := db.User{
		Name:             name,
		APIKeyHash:       utils.HashString(apiKey),
		Last4Digits:      apiKey[len(apiKey)-4:],
		AllowedProviders: providers,
		AllowedModels:    models,
		MaxCostMicros:    costLimit,
		Expiration:       expiration,
	}
	return apiKey, database.Create(&user).Error
}

func saveAllowedModels(database *gorm.DB, choices []modelChoice) ([]db.Provider, []db.Model, error) {
	var providers []db.Provider
	var models []db.Model
	seen := map[string]bool{}

	for _, choice := range choices {
		def := providerByName(choice.Provider)
		provider, err := ensureProvider(database, def)
		if err != nil {
			return nil, nil, err
		}
		model, err := ensureModel(database, provider.ID, choice.Model)
		if err != nil {
			return nil, nil, err
		}
		if !seen[provider.Name] {
			providers = append(providers, provider)
			seen[provider.Name] = true
		}
		models = append(models, model)
	}
	return providers, models, nil
}

func ensureProvider(database *gorm.DB, def providerDef) (db.Provider, error) {
	provider := db.Provider{Name: def.Name, BaseURL: def.BaseURL, EnvKey: def.EnvKey}
	err := database.Where("name = ?", def.Name).First(&provider).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return provider, database.Create(&provider).Error
	}
	if err == nil && provider.BaseURL != def.BaseURL {
		provider.BaseURL = def.BaseURL
		err = database.Save(&provider).Error
	}
	return provider, err
}

func ensureModel(database *gorm.DB, providerID uint, name string) (db.Model, error) {
	model := db.Model{ProviderID: providerID, Model: name}
	err := database.Where("provider_id = ? AND model = ?", providerID, name).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model, database.Create(&model).Error
	}
	return model, err
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk-" + base64.RawURLEncoding.EncodeToString(b), nil
}
