package api

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/jlevesy/prometheus-elector/election"
	"k8s.io/klog/v2"
)

type proxy struct {
	electionStatus election.Status

	localProxy http.Handler
	proxyCache proxyCache
}

func newProxy(cfg Config, electionStatus election.Status) (*proxy, error) {
	localProxy, err := newReverseProxy(
		fmt.Sprintf("http://localhost:%d", cfg.PrometheusLocalPort),
	)
	if err != nil {
		return nil, fmt.Errorf("could not build local instance reverse proxy: %w", err)
	}

	return &proxy{
		electionStatus: electionStatus,
		localProxy:     localProxy,
		proxyCache: proxyCache{
			proxies: make(map[string]http.Handler),
			newProxy: func(memberID string) (http.Handler, error) {
				return newReverseProxy(
					fmt.Sprintf("http://%s.%s:%d", memberID, cfg.PrometheusServiceName, cfg.PrometheusRemotePort),
				)
			},
		},
	}, nil
}

func (p *proxy) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if p.electionStatus.IsLeader() {
		p.localProxy.ServeHTTP(rw, r)
		return
	}

	proxy, err := p.proxyCache.findOrCreateProxy(p.electionStatus.GetLeader())
	if err != nil {
		klog.ErrorS(err, "unable to retrieve proxy")
		http.Error(rw, "Something unexpected happened", http.StatusInternalServerError)
		return
	}

	proxy.ServeHTTP(rw, r)
}

type proxyCache struct {
	proxiesMu sync.RWMutex
	proxies   map[string]http.Handler
	newProxy  func(string) (http.Handler, error)
}

func (p *proxyCache) findOrCreateProxy(memberID string) (http.Handler, error) {
	p.proxiesMu.RLock()
	proxy, ok := p.proxies[memberID]
	p.proxiesMu.RUnlock()

	if ok {
		return proxy, nil
	}

	p.proxiesMu.Lock()
	defer p.proxiesMu.Unlock()

	proxy, ok = p.proxies[memberID]
	if ok {
		return proxy, nil
	}

	var err error
	proxy, err = p.newProxy(memberID)
	if err != nil {
		return nil, fmt.Errorf("could not build new proxy for member %q, reason is: %w", memberID, err)
	}

	p.proxies[memberID] = proxy

	return proxy, nil
}

func newReverseProxy(addr string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	return httputil.NewSingleHostReverseProxy(url), nil
}
