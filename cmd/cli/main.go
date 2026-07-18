package main

import (
	"github.com/Kartik-2239/openai-proxy/internal/db"
)

func main() {
	DB, err := db.Open("proxy.db")
	if err != nil {
		panic(err)
	}

	// provider := db.Provider{
	// 	Name:    "openrouter",
	// 	BaseURL: "https://openrouter.ai/api/v1",
	// }
	// DB.Create(&provider) // provider.ID is now set

	// model := db.Model{
	// 	Model:      "google/gemma-4-26b-a4b-it",
	// 	ProviderID: provider.ID,
	// }
	// DB.Create(&model) // model.ID is now set

	// api_key := "32b888edbc10"
	// hash := utils.HashString(api_key)

	// user := db.User{
	// 	Name:             "example",
	// 	APIKeyHash:       hash,
	// 	Last4Digits:      api_key[len(api_key)-4:],
	// 	AllowedProviders: []db.Provider{provider},
	// 	AllowedModels:    []db.Model{model},
	// }
	// DB.Create(&user)
	var users []db.User
	DB.Preload("AllowedProviders").Preload("AllowedModels.Provider").Find(&users)

	for _, user := range users {
		println("User:", user.Name)
		println("  Allowed Providers:")
		for _, provider := range user.AllowedProviders {
			println("    -", provider.Name, "(", provider.BaseURL, ")")
		}
		println("  Allowed Models:")
		for _, model := range user.AllowedModels {
			println("    -", model.Model, "(", model.Provider.Name, ")")
		}
	}

}
