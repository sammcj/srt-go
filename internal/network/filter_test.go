package network

import (
	"testing"
)

func TestDomainFilter(t *testing.T) {
	tests := []struct {
		name          string
		defaultPolicy string
		allowed       []string
		denied        []string
		domain        string
		want          bool
	}{
		{
			name:          "exact match allowed",
			defaultPolicy: "deny",
			allowed:       []string{"example.com"},
			denied:        []string{},
			domain:        "example.com",
			want:          true,
		},
		{
			name:          "wildcard subdomain",
			defaultPolicy: "deny",
			allowed:       []string{"*.github.com"},
			denied:        []string{},
			domain:        "api.github.com",
			want:          true,
		},
		{
			name:          "not in allowlist with deny policy",
			defaultPolicy: "deny",
			allowed:       []string{"example.com"},
			denied:        []string{},
			domain:        "malicious.com",
			want:          false,
		},
		{
			name:          "denied takes precedence",
			defaultPolicy: "allow",
			allowed:       []string{"*.example.com"},
			denied:        []string{"bad.example.com"},
			domain:        "bad.example.com",
			want:          false,
		},
		{
			name:          "empty allowlist with deny policy denies all",
			defaultPolicy: "deny",
			allowed:       []string{},
			denied:        []string{},
			domain:        "example.com",
			want:          false,
		},
		{
			name:          "empty allowlist with allow policy allows all",
			defaultPolicy: "allow",
			allowed:       []string{},
			denied:        []string{},
			domain:        "example.com",
			want:          true,
		},
		{
			name:          "allow policy with denylist blocks specific domain",
			defaultPolicy: "allow",
			allowed:       []string{},
			denied:        []string{"bad.com"},
			domain:        "bad.com",
			want:          false,
		},
		{
			name:          "allow policy with denylist allows other domains",
			defaultPolicy: "allow",
			allowed:       []string{},
			denied:        []string{"bad.com"},
			domain:        "good.com",
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewDomainFilter(tt.defaultPolicy, tt.allowed, tt.denied)
			if err != nil {
				t.Fatalf("NewDomainFilter() error = %v", err)
			}

			got := filter.IsAllowed(tt.domain)
			if got != tt.want {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.domain, got, tt.want)
			}
		})
	}
}

func TestNormaliseDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Example.Com", "example.com"},
		{"example.com:443", "example.com"},
		{"GITHUB.COM:8080", "github.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normaliseDomain(tt.input)
			if got != tt.want {
				t.Errorf("normaliseDomain(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
