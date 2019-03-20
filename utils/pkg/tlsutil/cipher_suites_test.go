package tlsutil

import (
	"go/importer"
	"reflect"
	"strings"
	"testing"
)

func TestGetCipherSuites(t *testing.T) {
	pkg, err := importer.For("source", nil).Import("crypto/tls")
	if err != nil {
		t.Fatal(err)
	}
	cm := make(map[string]uint16)
	for _, s := range pkg.Scope().Names() {
		if strings.HasPrefix(s, "TLS_RSA_") || strings.HasPrefix(s, "TLS_ECDHE_") {
			v, ok := GetCipherSuite(s)
			if !ok {
				t.Fatalf("Go implements missing cipher suite %q (%v)", s, v)
			}
			cm[s] = v
		}
	}
	if !reflect.DeepEqual(cm, cipherSuites) {
		t.Fatalf("found unmatched cipher suites %v (Go) != %v", cm, cipherSuites)
	}
}
