package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Kartik-2239/pinwheel/internal/db"
	"github.com/Kartik-2239/pinwheel/internal/utils"
)

func AuthenticateRequest(r *http.Request, store *db.Store) (*db.User, error) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}
	user, err := store.GetUserByHash(r.Context(), utils.HashString(token))
	if err != nil {
		return nil, err
	}
	return user, nil
}
