package testutil

import (
	"net/http"
	"sync"
)

type PauseableHandler struct {
	Next   http.Handler
	mu     sync.Mutex
	paused bool
}

func (ph *PauseableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ph.mu.Lock()
	paused := ph.paused
	ph.mu.Unlock()
	if !paused {
		ph.Next.ServeHTTP(w, r)
	} else {
		hj, ok := w.(http.Hijacker)
		if !ok {
			panic("webserver doesn't support hijacking")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			panic(err.Error())
		}
		conn.Close()
	}
}

func (ph *PauseableHandler) Pause() {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.paused = true
}

func (ph *PauseableHandler) Resume() {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.paused = false
}
