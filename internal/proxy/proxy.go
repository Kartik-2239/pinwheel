package proxy

import (
	"encoding/json"
	"fmt"
	"io"
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
	// r.Header.Set()
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
			baseurl, err := store.GetBaseURLForModel(context, model, pr.In.Header.Get("Authorization"))
			if err != nil {
				pr.Out.Body = io.NopCloser(pr.In.Body)
				return
			}
			u, _ := url.Parse(baseurl)
			pr.Out.Body = io.NopCloser(strings.NewReader(string(body)))
			pr.Out.ContentLength = int64(len(body))
			pr.Out.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
			pr.SetURL(u)
			pr.Out.Host = u.Host
			pr.Out.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("API_KEY")))
		},
	}
	return p
}
