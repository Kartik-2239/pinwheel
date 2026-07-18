package auth

import (
	"fmt"
	"strings"

	"github.com/Kartik-2239/openai-proxy/internal/db"
)

func IsModelAllowed(user *db.User, model string) (string, bool, error) {

	for _, allowedModel := range user.AllowedModels {
		if strings.EqualFold(allowedModel.Model, model) {
			return fmt.Sprintf("%s/%s", allowedModel.Provider.Name, allowedModel.Model), true, nil
		}
	}
	return "", false, nil
}
