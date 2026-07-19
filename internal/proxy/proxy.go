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

	"github.com/Kartik-2239/openai-proxy/internal/db"
	"github.com/joho/godotenv"
)

func New(store *db.Store) *httputil.ReverseProxy {
	err := godotenv.Load()
	if err != nil {
		return nil
	}
	var modeltop string
	var apiKeyTop string

	p := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			body, err := io.ReadAll(pr.In.Body)
			if err != nil {
				pr.Out.Body = io.NopCloser(pr.In.Body)
				return
			}
			var v map[string]any
			json.Unmarshal(body, &v)
			context := pr.In.Context()
			model, ok := v["model"].(string)
			if !ok {
				pr.Out.Body = io.NopCloser(pr.In.Body)
				return
			}
			newModel, err := store.GetModelFromName(context, model) //, pr.In.Header.Get("Authorization"))
			if err != nil {
				pr.Out.Body = io.NopCloser(pr.In.Body)
				return
			}
			v["model"] = newModel.Model
			modeltop = newModel.Model
			apiKeyTop = pr.In.Header.Get("Authorization")
			body, _ = json.Marshal(v)
			u, _ := url.Parse(newModel.Provider.BaseURL)
			pr.Out.Body = io.NopCloser(strings.NewReader(string(body)))
			pr.Out.ContentLength = int64(len(body))
			pr.Out.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
			pr.SetURL(u)
			pr.Out.Host = u.Host
			pr.Out.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv(newModel.Provider.EnvKey)))
		},
		ModifyResponse: func(r *http.Response) error {
			r.Body = &transformReader{r: r.Body, store: store, model: modeltop, apiKey: apiKeyTop, ctx: r.Request.Context()}
			return nil
		},
	}
	return p
}

type usage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

type transformReader struct {
	r      io.ReadCloser
	buf    []byte
	l      string
	done   bool
	store  *db.Store
	model  string
	apiKey string
	ctx    context.Context
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
					usage := usage{
						PromptTokens:     int64(usageData["prompt_tokens"].(float64)),
						CompletionTokens: int64(usageData["completion_tokens"].(float64)),
						TotalTokens:      int64(usageData["total_tokens"].(float64)),
					}
					fmt.Printf("Usage: %+v\n", usage)
					tr.done = true

					if err := tr.store.CreateUsage(tr.ctx, tr.apiKey, tr.model, usage.PromptTokens, usage.CompletionTokens); err != nil {
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
