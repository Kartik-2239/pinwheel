package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/Kartik-2239/pinwheel/internal/db"
	"github.com/joho/godotenv"
)

type proxyContextKey string

const proxyCtxKey proxyContextKey = "proxyCtx"

type proxyCtx struct {
	apiKey   string
	model    string
	provider string
}

func New(store *db.Store) *httputil.ReverseProxy {
	godotenv.Load()

	p := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {},
		ModifyResponse: func(r *http.Response) error {

			proxyContext, ok := r.Request.Context().Value(proxyCtxKey).(proxyCtx)
			if !ok {
				return fmt.Errorf("proxy context not found")
			}
			apiKey := proxyContext.apiKey
			model := proxyContext.model
			provider := proxyContext.provider

			if !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "text/event-stream") {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					return err
				}
				r.Body.Close()

				err = ExtractAndStoreUsage(store, body, proxyContext, r.Request.Context())
				if err != nil {
					return err
				}

				// remake the body because io.ReadAll consumes it!
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
				err = ExtractAndStoreUsage(tr.store, []byte(tr.l), proxyCtx{apiKey: tr.apiKey, model: tr.model, provider: tr.provider}, tr.ctx)
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
