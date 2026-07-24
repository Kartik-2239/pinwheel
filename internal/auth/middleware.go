package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Kartik-2239/pinwheel/internal/db"
)

func Middleware(store *db.Store) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.URL)

			user, err := AuthenticateRequest(r, store)
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

			AuthorizeRequest(w, r, user, v, store)

			// remake the body because io.ReadAll uses it up
			r.Body = io.NopCloser(strings.NewReader(string(body)))
			r.ContentLength = int64(len(body))
			r.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))

			next.ServeHTTP(w, r)
		})
	}
}
