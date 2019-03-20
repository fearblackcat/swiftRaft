package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
)

// NewCertPool creates x509 certPool with provided CA files.
func NewCertPool(CAFiles []string) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()

	for _, CAFile := range CAFiles {
		pemByte, err := ioutil.ReadFile(CAFile)
		if err != nil {
			return nil, err
		}

		for {
			var block *pem.Block
			block, pemByte = pem.Decode(pemByte)
			if block == nil {
				break
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			certPool.AddCert(cert)
		}
	}

	return certPool, nil
}

// NewCert generates TLS cert by using the given cert,key and parse function.
func NewCert(certfile, keyfile string, parseFunc func([]byte, []byte) (tls.Certificate, error)) (*tls.Certificate, error) {
	cert, err := ioutil.ReadFile(certfile)
	if err != nil {
		return nil, err
	}

	key, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}

	if parseFunc == nil {
		parseFunc = tls.X509KeyPair
	}

	tlsCert, err := parseFunc(cert, key)
	if err != nil {
		return nil, err
	}
	return &tlsCert, nil
}
