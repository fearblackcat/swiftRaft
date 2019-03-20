package rafthttp

import (
	"errors"
	"net/http"
)

func (t *roundTripperBlocker) RoundTrip(req *http.Request) (*http.Response, error) {
	c := make(chan struct{}, 1)
	t.mu.Lock()
	t.cancel[req] = c
	t.mu.Unlock()
	ctx := req.Context()
	select {
	case <-t.unblockc:
		return &http.Response{StatusCode: http.StatusNoContent, Body: &nopReadCloser{}}, nil
	case <-req.Cancel:
		return nil, errors.New("request canceled")
	case <-ctx.Done():
		return nil, errors.New("request canceled")
	case <-c:
		return nil, errors.New("request canceled")
	}
}
