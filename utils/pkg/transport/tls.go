package transport

import (
	"fmt"
	"strings"
	"time"
)

// ValidateSecureEndpoints scans the given endpoints against tls info, returning only those
// endpoints that could be validated as secure.
func ValidateSecureEndpoints(tlsInfo TLSInfo, eps []string) ([]string, error) {
	t, err := NewTransport(tlsInfo, 5*time.Second)
	if err != nil {
		return nil, err
	}
	var errs []string
	var endpoints []string
	for _, ep := range eps {
		if !strings.HasPrefix(ep, "https://") {
			errs = append(errs, fmt.Sprintf("%q is insecure", ep))
			continue
		}
		conn, cerr := t.Dial("tcp", ep[len("https://"):])
		if cerr != nil {
			errs = append(errs, fmt.Sprintf("%q failed to dial (%v)", ep, cerr))
			continue
		}
		conn.Close()
		endpoints = append(endpoints, ep)
	}
	if len(errs) != 0 {
		err = fmt.Errorf("%s", strings.Join(errs, ","))
	}
	return endpoints, err
}
