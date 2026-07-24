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

func ExtractAndStoreUsage(store *db.Store, body []byte, proxyContext proxyCtx, ctx context.Context) error {
	var v map[string]any
	err := json.Unmarshal(body, &v)
	if err != nil {
		return fmt.Errorf("CreateUsage error: %v\n", err)
	}
	if usageData, ok := v["usage"].(map[string]any); ok {
		costMicros := int64(0)
		if cost, ok := usageData["cost"].(float64); ok && cost > 0 {
			costMicros = int64(cost * 1e6)
		}
		PromptTokens, ok := usageData["prompt_tokens"].(float64)
		if !ok {
			PromptTokens = 0
		}
		CompletionTokens, ok := usageData["completion_tokens"].(float64)
		if !ok {
			CompletionTokens = 0
		}
		if err := store.CreateUsage(ctx, proxyContext.apiKey, proxyContext.model, proxyContext.provider, int64(PromptTokens), int64(CompletionTokens), &costMicros); err != nil {
			return fmt.Errorf("CreateUsage error: %v\n", err)
		}
	}
	return nil

}

func MakeAuthReq(req *http.Request, v map[string]any, newModel db.Model, originalAuth string, originalPath string, body []byte, baseCtx context.Context, roundTripper http.RoundTripper) (*http.Response, error) {
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
	proxyCtx := proxyCtx{
		apiKey:   originalAuth,
		model:    newModel.Model,
		provider: newModel.Provider.Name,
	}
	ctx := context.WithValue(baseCtx, proxyCtxKey, proxyCtx)
	req = req.WithContext(ctx)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv(newModel.Provider.EnvKey)))

	fmt.Println(req.URL.String())

	resp, err := roundTripper.RoundTrip(req)
	return resp, err
}
