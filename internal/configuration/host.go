package configuration

import (
	"fmt"
	"net"
	"strings"
)

// HostLiteral validates an ASCII DNS hostname or IP address without DNS lookup.
// It returns a URI authority host: IPv6 is bracketed. Schemes, ports, credentials,
// paths, scoped addresses and control characters are deliberately unsupported.
func HostLiteral(host string) (string, error) {
	raw := host
	bracketed := strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]")
	if bracketed {
		raw = host[1 : len(host)-1]
	}
	if ip := net.ParseIP(raw); ip != nil {
		if strings.Contains(raw, ":") {
			return "[" + ip.String() + "]", nil
		}
		if !bracketed {
			return ip.String(), nil
		}
	}
	fail := func() (string, error) {
		return "", fmt.Errorf("expected a DNS hostname, IPv4, or IPv6 address without scheme, port, credentials, or path")
	}
	if len(host) == 0 || len(host) > 253 || bracketed {
		return fail()
	}
	domain := strings.TrimSuffix(host, ".")
	numeric := true
	for _, r := range domain {
		if (r < '0' || r > '9') && r != '.' {
			numeric = false
		}
	}
	if numeric {
		return fail()
	}
	for _, label := range strings.Split(domain, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return fail()
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
				return fail()
			}
		}
	}
	return host, nil
}
