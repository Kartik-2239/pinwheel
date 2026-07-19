package cli

import (
	"os"

	"github.com/joho/godotenv"
)

type providerDef struct {
	Name      string
	BaseURL   string
	ModelsURL string
	EnvKey    string
}

var providerDefs = []providerDef{
	{Name: "openrouter", BaseURL: "https://openrouter.ai/api/v1", ModelsURL: "https://openrouter.ai/api/v1/models", EnvKey: "OPENROUTER_API_KEY"},
	{Name: "openai", BaseURL: "https://api.openai.com/v1", ModelsURL: "https://api.openai.com/v1/models", EnvKey: "OPENAI_API_KEY"},
	{Name: "anthropic", BaseURL: "https://api.anthropic.com/v1", ModelsURL: "https://api.anthropic.com/v1/models", EnvKey: "ANTHROPIC_API_KEY"},
}

func loadProviders() []providerDef {
	_ = godotenv.Load()
	var providers []providerDef
	for _, def := range providerDefs {
		if os.Getenv(def.EnvKey) != "" {
			providers = append(providers, def)
		}
	}
	return providers
}

func providerByName(name string) providerDef {
	for _, def := range providerDefs {
		if def.Name == name {
			return def
		}
	}
	return providerDef{}
}
