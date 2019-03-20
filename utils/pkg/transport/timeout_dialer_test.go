package transport

import (
	"net"
	"testing"
	"time"
)

func TestReadWriteTimeoutDialer(t *testing.T) {
	stop := make(chan struct{})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}
	ts := testBlockingServer{ln, 2, stop}
	go ts.Start(t)

	d := rwTimeoutDialer{
		wtimeoutd:  10 * time.Millisecond,
		rdtimeoutd: 10 * time.Millisecond,
	}
	conn, err := d.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("unexpected dial error: %v", err)
	}
	defer conn.Close()

	// fill the socket buffer
	data := make([]byte, 5*1024*1024)
	done := make(chan struct{})
	go func() {
		_, err = conn.Write(data)
		done <- struct{}{}
	}()

	select {
	case <-done:
	// Wait 5s more than timeout to avoid delay in low-end systems;
	// the slack was 1s extra, but that wasn't enough for CI.
	case <-time.After(d.wtimeoutd*10 + 5*time.Second):
		t.Fatal("wait timeout")
	}

	if operr, ok := err.(*net.OpError); !ok || operr.Op != "write" || !operr.Timeout() {
		t.Errorf("err = %v, want write i/o timeout error", err)
	}

	conn, err = d.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("unexpected dial error: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 10)
	go func() {
		_, err = conn.Read(buf)
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(d.rdtimeoutd * 10):
		t.Fatal("wait timeout")
	}

	if operr, ok := err.(*net.OpError); !ok || operr.Op != "read" || !operr.Timeout() {
		t.Errorf("err = %v, want write i/o timeout error", err)
	}

	stop <- struct{}{}
}

type testBlockingServer struct {
	ln   net.Listener
	n    int
	stop chan struct{}
}

func (ts *testBlockingServer) Start(t *testing.T) {
	for i := 0; i < ts.n; i++ {
		conn, err := ts.ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
	}
	<-ts.stop
}
