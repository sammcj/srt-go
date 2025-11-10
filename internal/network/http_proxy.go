package network

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// HTTPProxy is an HTTP/HTTPS proxy server with domain filtering
type HTTPProxy struct {
	port     int
	filter   *DomainFilter
	server   *http.Server
	listener net.Listener
}

// NewHTTPProxy creates a new HTTP proxy
func NewHTTPProxy(filter *DomainFilter, port int) (*HTTPProxy, error) {
	proxy := &HTTPProxy{
		port:   port,
		filter: filter,
	}

	// Create listener
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	// Get actual port if auto-assigned
	if port == 0 {
		proxy.port = listener.Addr().(*net.TCPAddr).Port
	}

	proxy.listener = listener

	// Create server
	proxy.server = &http.Server{
		Handler:      http.HandlerFunc(proxy.handleRequest),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return proxy, nil
}

// Port returns the proxy port
func (p *HTTPProxy) Port() int {
	return p.port
}

// Start starts the proxy server
func (p *HTTPProxy) Start() error {
	slog.Debug("HTTP proxy starting", "port", p.port)
	return p.server.Serve(p.listener)
}

// Stop stops the proxy server
func (p *HTTPProxy) Stop() error {
	return p.server.Close()
}

func (p *HTTPProxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Extract domain
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	domain := strings.Split(host, ":")[0]

	// Check filter
	if !p.filter.IsAllowed(domain) {
		slog.Debug("HTTP proxy blocked request", "domain", domain, "method", r.Method)
		w.Header().Set("X-Proxy-Error", "blocked-by-allowlist")
		http.Error(w, "Domain not allowed by sandbox policy", http.StatusForbidden)
		return
	}

	// Handle CONNECT for HTTPS
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
		return
	}

	// Handle regular HTTP
	p.handleHTTP(w, r)
}

func (p *HTTPProxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	// Hijack the connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	// Connect to target
	targetConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		slog.Debug("HTTP proxy failed to connect", "host", r.Host, "error", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer targetConn.Close()

	// Send 200 Connection Established
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Tunnel traffic bidirectionally
	go io.Copy(targetConn, clientConn)
	io.Copy(clientConn, targetConn)
}

func (p *HTTPProxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Create client request
	targetURL := r.URL
	if targetURL.Scheme == "" {
		targetURL.Scheme = "http"
	}
	if targetURL.Host == "" {
		targetURL.Host = r.Host
	}

	// Create new request
	proxyReq, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy headers
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	// Remove hop-by-hop headers
	removeHopByHopHeaders(proxyReq.Header)

	// Send request
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		slog.Debug("HTTP proxy request failed", "url", targetURL.String(), "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Remove hop-by-hop headers
	removeHopByHopHeaders(w.Header())

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy body
	io.Copy(w, resp.Body)
}

func removeHopByHopHeaders(h http.Header) {
	// Remove hop-by-hop headers as per RFC 2616
	h.Del("Connection")
	h.Del("Keep-Alive")
	h.Del("Proxy-Authenticate")
	h.Del("Proxy-Authorization")
	h.Del("Te")
	h.Del("Trailers")
	h.Del("Transfer-Encoding")
	h.Del("Upgrade")
}
