package steps

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"

	"webpage-analyzer/internal/pipeline"
	"webpage-analyzer/internal/urlutil"
)

var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

var privateRanges []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"0.0.0.0/8",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	} {
		_, network, _ := net.ParseCIDR(cidr)
		privateRanges = append(privateRanges, network)
	}
}

// URLValidation is Stage 1. Validates the domain format and guards against SSRF.
type URLValidation struct{}

func (s *URLValidation) Name() string { return "url_validation" }
func (s *URLValidation) Stage() int   { return 1 }

func (s *URLValidation) Run(ctx context.Context, state *pipeline.State) error {
	domain := urlutil.Domain(state.URL)
	domain = strings.ToLower(domain)

	if !domainRegex.MatchString(domain) {
		err := fmt.Errorf("invalid domain format: %q", domain)
		state.SetResult(s.Name(), pipeline.StepResult{Status: "failed", Error: err.Error()})
		return err
	}

	addrs, err := net.DefaultResolver.LookupHost(ctx, domain)
	if err != nil {
		e := fmt.Errorf("could not resolve domain %q", domain)
		state.SetResult(s.Name(), pipeline.StepResult{Status: "failed", Error: e.Error()})
		return e
	}

	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip != nil && isPrivateIP(ip) {
			e := fmt.Errorf("domain resolves to a private IP address — request blocked")
			state.SetResult(s.Name(), pipeline.StepResult{Status: "failed", Error: e.Error()})
			return e
		}
	}

	state.SetResult(s.Name(), pipeline.StepResult{Status: "done"})
	return nil
}

func isPrivateIP(ip net.IP) bool {
	for _, network := range privateRanges {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
