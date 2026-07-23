package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/Kartik-2239/pinwheel/internal/db"
	"github.com/joho/godotenv"
)

type proxyContextKey string

const (
	ctxAPIKey   proxyContextKey = "apiKey"
	ctxModel    proxyContextKey = "model"
	ctxProvider proxyContextKey = "provider"
)

func New(store *db.Store) *httputil.ReverseProxy {
	godotenv.Load()
	// if err != nil {
	// 	return nil
	// }

	p := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			// body, err := io.ReadAll(pr.In.Body)
			// pr.Out.URL.Scheme = pr.In.URL.Scheme
			// if err != nil {
			// 	pr.Out.Body = io.NopCloser(pr.In.Body)
			// 	return
			// }
			// var v map[string]any
			// json.Unmarshal(body, &v)
			// incontext := pr.In.Context()
			// model, ok := v["model"].(string)
			// if !ok {
			// 	pr.Out.Body = io.NopCloser(pr.In.Body)
			// 	return
			// }
			// newModel, err := Router(store, incontext, model, pr.In.Header.Get("Authorization"))
			// if err != nil {
			// 	pr.Out.Body = io.NopCloser(pr.In.Body)
			// 	return
			// }
			// v["model"] = newModel.Model
			// apiKey := pr.In.Header.Get("Authorization")

			// ctx := context.WithValue(pr.Out.Context(), ctxAPIKey, apiKey)
			// ctx = context.WithValue(ctx, ctxModel, newModel.Model)
			// pr.Out = pr.Out.WithContext(ctx)

			// body, _ = json.Marshal(v)
			// baseURL := newModel.Provider.BaseURL
			// u, _ := url.Parse(baseURL)
			// pr.Out.Body = io.NopCloser(strings.NewReader(string(body)))
			// pr.Out.ContentLength = int64(len(body))
			// pr.Out.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
			// pr.SetURL(u)
			// pr.Out.Host = u.Host
			// pr.Out.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv(newModel.Provider.EnvKey)))
		},
		ModifyResponse: func(r *http.Response) error {
			apiKey, _ := r.Request.Context().Value(ctxAPIKey).(string)
			model, _ := r.Request.Context().Value(ctxModel).(string)
			provider, _ := r.Request.Context().Value(ctxProvider).(string)

			if !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "text/event-stream") {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					return err
				}
				r.Body.Close()

				var v map[string]any
				if err := json.Unmarshal(body, &v); err == nil {
					if usageData, ok := v["usage"].(map[string]any); ok {
						costMicros := int64(0)
						if cost, ok := usageData["cost"].(float64); ok && cost > 0 {
							costMicros = int64(cost * 1e6)
						}
						usage := usage{
							PromptTokens:     int64(usageData["prompt_tokens"].(float64)),
							CompletionTokens: int64(usageData["completion_tokens"].(float64)),
							TotalTokens:      int64(usageData["total_tokens"].(float64)),
						}
						if err := store.CreateUsage(r.Request.Context(), apiKey, model, provider, usage.PromptTokens, usage.CompletionTokens, &costMicros); err != nil {
							fmt.Printf("CreateUsage error: %v\n", err)
						}
					}
				}
				r.Body = io.NopCloser(bytes.NewReader(body))
				r.ContentLength = int64(len(body))
				r.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
				return nil
			}
			r.Body = &transformReader{r: r.Body, store: store, model: model, provider: provider, apiKey: apiKey, ctx: r.Request.Context()}
			return nil
		},
		Transport: &transport{base: http.DefaultTransport, store: store},
	}
	return p
}

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

type usage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

type transformReader struct {
	r        io.ReadCloser
	buf      []byte
	l        string
	done     bool
	store    *db.Store
	model    string
	apiKey   string
	ctx      context.Context
	provider string
}

func (tr *transformReader) Read(p []byte) (n int, err error) {
	n, err = tr.r.Read(p)
	if n > 0 {
		tr.buf = append(tr.buf, p[:n]...)
		for {
			idx := bytes.IndexByte(tr.buf, '\n')
			if idx == -1 {
				break
			}
			line := bytes.TrimSpace(tr.buf[:idx])
			tr.buf = tr.buf[idx+1:]

			line = bytes.TrimPrefix(line, []byte("data: "))
			if len(line) == 0 || string(line) == "[DONE]" {
				if tr.done {
					continue
				}
				var v map[string]any
				json.Unmarshal([]byte(tr.l), &v)
				if usageData, ok := v["usage"].(map[string]any); ok {
					cost, ok := usageData["cost"].(float64)
					var costmicros *int64
					if ok && cost > 0 {
						c := int64(cost * 1e6)
						costmicros = &c
					}
					usage := usage{
						PromptTokens:     int64(usageData["prompt_tokens"].(float64)),
						CompletionTokens: int64(usageData["completion_tokens"].(float64)),
						TotalTokens:      int64(usageData["total_tokens"].(float64)),
					}
					// fmt.Printf("Usage: %+v\n", usage)
					tr.done = true

					if err := tr.store.CreateUsage(tr.ctx, tr.apiKey, tr.model, tr.provider, usage.PromptTokens, usage.CompletionTokens, costmicros); err != nil {
						fmt.Printf("CreateUsage error: %v\n", err)
					}

				}
				continue
			}
			tr.l = string(line)
		}
	}
	return n, err
}

func (tr *transformReader) Close() error {
	return tr.r.Close()
}
