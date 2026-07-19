package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Kartik-2239/openai-proxy/internal/db"
	"github.com/Kartik-2239/openai-proxy/internal/utils"
)

// curl https://<base_url>/chat/completions \
//   -H "Content-Type: application/json" \
//   -H "Authorization: Bearer $OPENAI_API_KEY" \
//   -d '{
//     "model": "VAR_chat_model_id",
//     "messages": [
//       {
//         "role": "developer",
//         "content": "You are a helpful assistant."
//       },
//       {
//         "role": "user",
//         "content": "Hello!"
//       }
//     ]
//   }'

func Middleware(store *db.Store) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.URL)
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			user, err := store.GetUserByHash(r.Context(), utils.HashString(token))
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var v map[string]any
			json.Unmarshal(body, &v)

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

			// for key, value := range v {
			// 	fmt.Printf("%s: %v\n", key, value)
			// }
			r.Body = io.NopCloser(strings.NewReader(string(body)))
			r.ContentLength = int64(len(body))
			r.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))

			next.ServeHTTP(w, r)
		})
	}
}
