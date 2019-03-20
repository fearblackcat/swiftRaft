package transport

import (
	"crypto/tls"
	"net"
	"net/http"
	"testing"
)

// TestNewKeepAliveListener tests NewKeepAliveListener returns a listener
// that accepts connections.
// TODO: verify the keepalive option is set correctly
func TestNewKeepAliveListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}

	ln, err = NewKeepAliveListener(ln, "http", nil)
	if err != nil {
		t.Fatalf("unexpected NewKeepAliveListener error: %v", err)
	}

	go http.Get("http://" + ln.Addr().String())
	conn, err := ln.Accept()
	if err != nil {
		t.Fatalf("unexpected Accept error: %v", err)
	}
	conn.Close()
	ln.Close()

	ln, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("unexpected Listen error: %v", err)
	}

	// tls
	tlsinfo, del, err := createSelfCert()
	if err != nil {
		t.Fatalf("unable to create tmpfile: %v", err)
	}
	defer del()
	tlsInfo := TLSInfo{CertFile: tlsinfo.CertFile, KeyFile: tlsinfo.KeyFile}
	tlsInfo.parseFunc = fakeCertificateParserFunc(tls.Certificate{}, nil)
	tlscfg, err := tlsInfo.ServerConfig()
	if err != nil {
		t.Fatalf("unexpected serverConfig error: %v", err)
	}
	tlsln, err := NewKeepAliveListener(ln, "https", tlscfg)
	if err != nil {
		t.Fatalf("unexpected NewKeepAliveListener error: %v", err)
	}

	go http.Get("https://" + tlsln.Addr().String())
	conn, err = tlsln.Accept()
	if err != nil {
		t.Fatalf("unexpected Accept error: %v", err)
	}
	if _, ok := conn.(*tls.Conn); !ok {
		t.Errorf("failed to accept *tls.Conn")
	}
	conn.Close()
	tlsln.Close()
}

func TestNewKeepAliveListenerTLSEmptyConfig(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}

	_, err = NewKeepAliveListener(ln, "https", nil)
	if err == nil {
		t.Errorf("err = nil, want not presented error")
	}
}
