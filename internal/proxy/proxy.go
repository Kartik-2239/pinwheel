package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
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

	p := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {},
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
						PromptTokens, ok := usageData["prompt_tokens"].(float64)
						if !ok {
							PromptTokens = 0
						}
						CompletionTokens, ok := usageData["completion_tokens"].(float64)
						if !ok {
							CompletionTokens = 0
						}
						if err := store.CreateUsage(r.Request.Context(), apiKey, model, provider, int64(PromptTokens), int64(CompletionTokens), &costMicros); err != nil {
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
					PromptTokens, ok := usageData["prompt_tokens"].(float64)
					if !ok {
						PromptTokens = 0
					}
					CompletionTokens, ok := usageData["completion_tokens"].(float64)
					if !ok {
						CompletionTokens = 0
					}

					tr.done = true

					if err := tr.store.CreateUsage(tr.ctx, tr.apiKey, tr.model, tr.provider, int64(PromptTokens), int64(CompletionTokens), costmicros); err != nil {
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
