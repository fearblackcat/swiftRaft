package transport

import (
	"net"
	"time"
)

// NewTimeoutListener returns a listener that listens on the given address.
// If read/write on the accepted connection blocks longer than its time limit,
// it will return timeout error.
func NewTimeoutListener(addr string, scheme string, tlsinfo *TLSInfo, rdtimeoutd, wtimeoutd time.Duration) (net.Listener, error) {
	ln, err := newListener(addr, scheme)
	if err != nil {
		return nil, err
	}
	ln = &rwTimeoutListener{
		Listener:   ln,
		rdtimeoutd: rdtimeoutd,
		wtimeoutd:  wtimeoutd,
	}
	if ln, err = wrapTLS(scheme, tlsinfo, ln); err != nil {
		return nil, err
	}
	return ln, nil
}

type rwTimeoutListener struct {
	net.Listener
	wtimeoutd  time.Duration
	rdtimeoutd time.Duration
}

func (rwln *rwTimeoutListener) Accept() (net.Conn, error) {
	c, err := rwln.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return timeoutConn{
		Conn:       c,
		wtimeoutd:  rwln.wtimeoutd,
		rdtimeoutd: rwln.rdtimeoutd,
	}, nil
}
