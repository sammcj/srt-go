package network

import (
	"strings"

	"github.com/gobwas/glob"
)

// DomainFilter filters network connections by domain
type DomainFilter struct {
	allowed       []DomainPattern
	denied        []DomainPattern
	defaultPolicy string // "allow" or "deny"
}

// DomainPattern represents a domain matching pattern
type DomainPattern struct {
	pattern  string
	isGlob   bool
	compiled glob.Glob
}

// NewDomainFilter creates a new domain filter
func NewDomainFilter(defaultPolicy string, allowedDomains, deniedDomains []string) (*DomainFilter, error) {
	// Default to "allow" if not specified or invalid
	if defaultPolicy != "allow" && defaultPolicy != "deny" {
		defaultPolicy = "allow"
	}

	filter := &DomainFilter{
		allowed:       make([]DomainPattern, 0, len(allowedDomains)),
		denied:        make([]DomainPattern, 0, len(deniedDomains)),
		defaultPolicy: defaultPolicy,
	}

	// Compile allowed patterns
	for _, domain := range allowedDomains {
		pattern, err := compileDomainPattern(domain)
		if err != nil {
			return nil, err
		}
		filter.allowed = append(filter.allowed, pattern)
	}

	// Compile denied patterns
	for _, domain := range deniedDomains {
		pattern, err := compileDomainPattern(domain)
		if err != nil {
			return nil, err
		}
		filter.denied = append(filter.denied, pattern)
	}

	return filter, nil
}

// IsAllowed checks if a domain is allowed
func (f *DomainFilter) IsAllowed(domain string) bool {
	// Normalise domain (lowercase, strip port)
	domain = normaliseDomain(domain)

	// Check denied list first (deny takes precedence)
	for _, pattern := range f.denied {
		if pattern.Matches(domain) {
			return false
		}
	}

	// Check allowed list
	for _, pattern := range f.allowed {
		if pattern.Matches(domain) {
			return true
		}
	}

	// Use default policy if no match
	return f.defaultPolicy == "allow"
}

// Matches checks if a domain matches this pattern
func (p *DomainPattern) Matches(domain string) bool {
	domain = normaliseDomain(domain)

	if p.isGlob {
		return p.compiled.Match(domain)
	}

	// Exact match
	return domain == p.pattern
}

func compileDomainPattern(pattern string) (DomainPattern, error) {
	// Normalise
	pattern = strings.ToLower(strings.TrimSpace(pattern))

	// Check if it's a wildcard pattern
	if strings.Contains(pattern, "*") {
		compiled, err := glob.Compile(pattern)
		if err != nil {
			return DomainPattern{}, err
		}

		return DomainPattern{
			pattern:  pattern,
			isGlob:   true,
			compiled: compiled,
		}, nil
	}

	// Exact match pattern
	return DomainPattern{
		pattern: pattern,
		isGlob:  false,
	}, nil
}

func normaliseDomain(domain string) string {
	// Remove port if present
	if idx := strings.LastIndex(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}

	// Lowercase
	return strings.ToLower(strings.TrimSpace(domain))
}
