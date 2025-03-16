package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

type Proxy struct {
	target    *url.URL
	proxy     *httputil.ReverseProxy
	timeoutMS int
}

func NewProxy(targetURL string, timeoutMS int) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	transport, err := createTransport(target, time.Duration(timeoutMS)*time.Millisecond)
	if err != nil {
		return nil, err
	}

	p := &Proxy{
		target:    target,
		timeoutMS: timeoutMS,
	}

	p.proxy = &httputil.ReverseProxy{
		Director:  p.director,
		Transport: transport,
	}

	return p, nil
}

func createTransport(target *url.URL, timeout time.Duration) (http.RoundTripper, error) {
	switch target.Scheme {
	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(target, proxy.Direct)
		if err != nil {
			return nil, err
		}
		return &http.Transport{
			Dial:                  dialer.Dial,
			ResponseHeaderTimeout: timeout,
		}, nil
	default: // 处理 http/https 和其他协议
		return &http.Transport{
			Proxy:                 http.ProxyURL(target),
			ResponseHeaderTimeout: timeout,
		}, nil
	}
}

func (p *Proxy) director(req *http.Request) {
	// 保留原始目标信息以便调试
	req.Header.Set("X-Proxy-Target", p.target.String())
	
	// 修正请求目标
	req.URL.Scheme = p.target.Scheme
	if p.target.Scheme == "socks5" || p.target.Scheme == "socks5h" {
		// 对于 SOCKS5 代理，实际请求目标需要从原始请求获取
		req.URL.Scheme = "http" // 目标服务的实际协议
	}
	req.URL.Host = p.target.Host
	req.Host = p.target.Host
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}
