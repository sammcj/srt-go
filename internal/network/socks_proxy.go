package network

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/armon/go-socks5"
)

// SOCKSProxy is a SOCKS5 proxy server with domain filtering
type SOCKSProxy struct {
	port     int
	filter   *DomainFilter
	server   *socks5.Server
	listener net.Listener
}

// NewSOCKSProxy creates a new SOCKS5 proxy
func NewSOCKSProxy(filter *DomainFilter, port int) (*SOCKSProxy, error) {
	proxy := &SOCKSProxy{
		port:   port,
		filter: filter,
	}

	// Create SOCKS5 config
	conf := &socks5.Config{
		Rules: &domainRuleSet{filter: filter},
	}

	server, err := socks5.New(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 server: %w", err)
	}

	proxy.server = server

	// Create listener immediately (like HTTP proxy does)
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

	return proxy, nil
}

// Port returns the proxy port
func (p *SOCKSProxy) Port() int {
	return p.port
}

// Start starts the proxy server
func (p *SOCKSProxy) Start() error {
	slog.Debug("SOCKS5 proxy starting", "port", p.port)
	return p.server.Serve(p.listener)
}

// Stop stops the proxy server
func (p *SOCKSProxy) Stop() error {
	if p.listener != nil {
		return p.listener.Close()
	}
	return nil
}

// domainRuleSet implements SOCKS5 rules for domain filtering
type domainRuleSet struct {
	filter *DomainFilter
}

// Allow checks if a SOCKS5 request should be allowed
func (r *domainRuleSet) Allow(ctx context.Context, req *socks5.Request) (context.Context, bool) {
	// Extract domain from request
	domain := ""

	if req.DestAddr != nil {
		if req.DestAddr.FQDN != "" {
			domain = req.DestAddr.FQDN
		} else if req.DestAddr.IP != nil {
			domain = req.DestAddr.IP.String()
		}
	}

	// Check filter
	allowed := r.filter.IsAllowed(domain)

	if !allowed {
		slog.Debug("SOCKS5 proxy blocked request", "domain", domain)
	}

	return ctx, allowed
}
