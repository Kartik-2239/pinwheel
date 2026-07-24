package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Kartik-2239/pinwheel/internal/db"
)

func AuthorizeRequest(w http.ResponseWriter, r *http.Request, user *db.User, v map[string]interface{}, store *db.Store) {
	if user.Expiration != nil && user.Expiration.Before(time.Now()) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.MaxCostMicros != nil {
		totalCostMicro, err := store.GetTotalCost(r.Context(), user.ID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if totalCostMicro >= *user.MaxCostMicros {
			http.Error(w, "Forbidden: Limit exceeded", http.StatusForbidden)
			return
		}
	}

	model, isModel, err := IsModelAllowed(user, v["model"].(string))
	fmt.Println("Model", model, isModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !isModel || model == "" {
		http.Error(w, "Forbidden Model", http.StatusForbidden)
		return
	}
}
