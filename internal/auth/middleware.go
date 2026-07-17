package auth

import (
	"net/http"
	"strings"

	"github.com/Kartik-2239/openai-proxy/internal/db"
	"github.com/Kartik-2239/openai-proxy/internal/utils"
)

func Middleware(store *db.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if _, err := store.GetUserByHash(r.Context(), utils.HashString(token)); err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
