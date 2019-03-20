package transport

import (
	"crypto/tls"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestNewTransportTLSInvalidCipherSuites expects a client with invalid
// cipher suites fail to handshake with the server.
func TestNewTransportTLSInvalidCipherSuites(t *testing.T) {
	tlsInfo, del, err := createSelfCert()
	if err != nil {
		t.Fatalf("unable to create cert: %v", err)
	}
	defer del()

	cipherSuites := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	}

	// make server and client have unmatched cipher suites
	srvTLS, cliTLS := *tlsInfo, *tlsInfo
	srvTLS.CipherSuites, cliTLS.CipherSuites = cipherSuites[:2], cipherSuites[2:]

	ln, err := NewListener("127.0.0.1:0", "https", &srvTLS)
	if err != nil {
		t.Fatalf("unexpected NewListener error: %v", err)
	}
	defer ln.Close()

	donec := make(chan struct{})
	go func() {
		ln.Accept()
		donec <- struct{}{}
	}()
	go func() {
		tr, err := NewTransport(cliTLS, 3*time.Second)
		if err != nil {
			t.Fatalf("unexpected NewTransport error: %v", err)
		}
		cli := &http.Client{Transport: tr}
		_, gerr := cli.Get("https://" + ln.Addr().String())
		if gerr == nil || !strings.Contains(gerr.Error(), "tls: handshake failure") {
			t.Fatal("expected client TLS handshake error")
		}
		ln.Close()
		donec <- struct{}{}
	}()
	<-donec
	<-donec
}
