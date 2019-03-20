// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package httputil provides HTTP utility functions.
package httputil

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
)

// GracefulClose drains http.Response.Body until it hits EOF
// and closes it. This prevents TCP/TLS connections from closing,
// therefore available for reuse.
// Borrowed from golang/net/context/ctxhttp/cancelreq.go.
func GracefulClose(resp *http.Response) {
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
}

// GetHostname returns the hostname from request Host field.
// It returns empty string, if Host field contains invalid
// value (e.g. "localhost:::" with too many colons).
func GetHostname(req *http.Request) string {
	if req == nil {
		return ""
	}
	h, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		return req.Host
	}
	return h
}
