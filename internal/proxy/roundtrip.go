package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Kartik-2239/pinwheel/internal/db"
)

type transport struct {
	base  http.RoundTripper
	store *db.Store
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Println("==========================")
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	var v map[string]any
	json.Unmarshal(body, &v)
	models, err := t.store.GetModelFromName(req.Context(), v["model"].(string), req.Header.Get("Authorization"))
	fmt.Println(models)
	if err != nil {
		return nil, err
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("model not found")
	}
	originalPath := req.URL.Path
	originalAuth := req.Header.Get("Authorization")
	baseCtx := req.Context()
	for _, newModel := range models {
		fmt.Println("Trying models", newModel.Model, newModel.Provider)
		resp, err := MakeAuthReq(req, v, newModel, originalAuth, originalPath, body, baseCtx, t.base)
		if resp.StatusCode == 200 {
			fmt.Println(resp.Status)
			fmt.Println(resp.Header.Values("Content-Type"))
			fmt.Println("==========================")
			return resp, err
		}
		fmt.Println(resp.Status)
	}
	return nil, fmt.Errorf("no models available")
}
