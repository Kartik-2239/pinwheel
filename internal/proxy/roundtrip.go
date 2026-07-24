package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

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
		v["model"] = newModel.Model
		body, _ = json.Marshal(v)
		req.Body = io.NopCloser(strings.NewReader(string(body)))
		req.ContentLength = int64(len(body))
		req.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))

		baseURL := strings.TrimRight(newModel.Provider.BaseURL, "/")
		u, err := url.Parse(baseURL)
		if err != nil {
			return nil, err
		}

		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host
		req.URL.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(originalPath, "/")
		req.Host = u.Host
		ctx := context.WithValue(baseCtx, ctxAPIKey, originalAuth)
		ctx = context.WithValue(ctx, ctxModel, newModel.Model)
		ctx = context.WithValue(ctx, ctxProvider, newModel.Provider.Name)
		req = req.WithContext(ctx)

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv(newModel.Provider.EnvKey)))

		fmt.Println(req.URL.String())

		resp, err := t.base.RoundTrip(req)
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
