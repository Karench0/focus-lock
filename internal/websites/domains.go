package websites

import (
	"errors"
	"net"
	"strings"

	"golang.org/x/net/idna"
)

// Normalize accepts a hostname, never a URL, wildcard, path or port.
func Normalize(input string) (string, error) {
	s := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(input)), ".")
	if strings.ContainsAny(s, "/:?#@*\\ \t\r\n") || s == "" {
		return "", errors.New("enter a domain, without protocol, path or port")
	}
	a, err := idna.Lookup.ToASCII(s)
	if err != nil || len(a) > 253 {
		return "", errors.New("invalid domain")
	}
	if a == "localhost" {
		return a, nil
	}
	if ip := net.ParseIP(a); ip != nil {
		if ip.IsLoopback() {
			return a, nil
		}
		return "", errors.New("only loopback IP addresses are supported")
	}
	labels := strings.Split(a, ".")
	if len(labels) < 2 {
		return "", errors.New("domain must contain a dot")
	}
	for _, l := range labels {
		if len(l) == 0 || len(l) > 63 || l[0] == '-' || l[len(l)-1] == '-' {
			return "", errors.New("invalid domain label")
		}
		for _, r := range l {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				return "", errors.New("invalid domain character")
			}
		}
	}
	return a, nil
}
