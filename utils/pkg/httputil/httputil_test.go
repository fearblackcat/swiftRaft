package httputil

import (
	"net/http"
	"testing"
)

func TestGetHostname(t *testing.T) {
	tt := []struct {
		req  *http.Request
		host string
	}{
		{&http.Request{Host: "localhost"}, "localhost"},
		{&http.Request{Host: "localhost:2379"}, "localhost"},
		{&http.Request{Host: "localhost."}, "localhost."},
		{&http.Request{Host: "localhost.:2379"}, "localhost."},
		{&http.Request{Host: "127.0.0.1"}, "127.0.0.1"},
		{&http.Request{Host: "127.0.0.1:2379"}, "127.0.0.1"},

		{&http.Request{Host: "localhos"}, "localhos"},
		{&http.Request{Host: "localhos:2379"}, "localhos"},
		{&http.Request{Host: "localhos."}, "localhos."},
		{&http.Request{Host: "localhos.:2379"}, "localhos."},
		{&http.Request{Host: "1.2.3.4"}, "1.2.3.4"},
		{&http.Request{Host: "1.2.3.4:2379"}, "1.2.3.4"},

		// too many colons in address
		{&http.Request{Host: "localhost:::::"}, "localhost:::::"},
	}
	for i := range tt {
		hv := GetHostname(tt[i].req)
		if hv != tt[i].host {
			t.Errorf("#%d: %q expected host %q, got '%v'", i, tt[i].req.Host, tt[i].host, hv)
		}
	}
}
