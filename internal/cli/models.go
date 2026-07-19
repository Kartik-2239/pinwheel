package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type modelChoice struct {
	Provider string
	Model    string
}

func fetchModelsCmd(providers []providerDef) tea.Cmd {
	return func() tea.Msg {
		models, err := fetchModels(providers)
		return modelsMsg{models: models, err: err}
	}
}

func fetchModels(providers []providerDef) ([]modelChoice, error) {
	var models []modelChoice
	client := &http.Client{Timeout: 30 * time.Second}

	for _, def := range providers {
		providerModels, err := fetchProviderModels(client, def)
		if err != nil {
			return nil, err
		}
		models = append(models, providerModels...)
	}

	sort.Slice(models, func(i, j int) bool { return models[i].Provider+models[i].Model < models[j].Provider+models[j].Model })
	return models, nil
}

func fetchProviderModels(client *http.Client, def providerDef) ([]modelChoice, error) {
	req, _ := http.NewRequest(http.MethodGet, def.ModelsURL, nil)
	setAuthHeaders(req, def)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s models: %w", def.Name, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("fetch %s models: %s", def.Name, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse %s models: %w", def.Name, err)
	}

	models := make([]modelChoice, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		if id := first(item.ID, item.Name, item.DisplayName); id != "" {
			models = append(models, modelChoice{Provider: def.Name, Model: id})
		}
	}
	return models, nil
}

func setAuthHeaders(req *http.Request, def providerDef) {
	apiKey := os.Getenv(def.EnvKey)
	if def.Name == "anthropic" {
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
}
