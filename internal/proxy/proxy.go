package proxy

import (
	"net/http/httputil"
	"net/url"
)

func New(target string) *httputil.ReverseProxy {
	u, _ := url.Parse(target)
	p := httputil.NewSingleHostReverseProxy(u)

	return p
}
