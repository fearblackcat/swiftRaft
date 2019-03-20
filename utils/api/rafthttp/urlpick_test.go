package rafthttp

import (
	"net/url"
	"testing"

	"github.com/fearblackcat/smartRaft/utils/pkg/testutil"
)

// TestURLPickerPickTwice tests that pick returns a possible url,
// and always returns the same one.
func TestURLPickerPickTwice(t *testing.T) {
	picker := mustNewURLPicker(t, []string{"http://127.0.0.1:2380", "http://127.0.0.1:7001"})

	u := picker.pick()
	urlmap := map[url.URL]bool{
		{Scheme: "http", Host: "127.0.0.1:2380"}: true,
		{Scheme: "http", Host: "127.0.0.1:7001"}: true,
	}
	if !urlmap[u] {
		t.Errorf("url picked = %+v, want a possible url in %+v", u, urlmap)
	}

	// pick out the same url when calling pick again
	uu := picker.pick()
	if u != uu {
		t.Errorf("url picked = %+v, want %+v", uu, u)
	}
}

func TestURLPickerUpdate(t *testing.T) {
	picker := mustNewURLPicker(t, []string{"http://127.0.0.1:2380", "http://127.0.0.1:7001"})
	picker.update(testutil.MustNewURLs(t, []string{"http://localhost:2380", "http://localhost:7001"}))

	u := picker.pick()
	urlmap := map[url.URL]bool{
		{Scheme: "http", Host: "localhost:2380"}: true,
		{Scheme: "http", Host: "localhost:7001"}: true,
	}
	if !urlmap[u] {
		t.Errorf("url picked = %+v, want a possible url in %+v", u, urlmap)
	}
}

func TestURLPickerUnreachable(t *testing.T) {
	picker := mustNewURLPicker(t, []string{"http://127.0.0.1:2380", "http://127.0.0.1:7001"})
	u := picker.pick()
	picker.unreachable(u)

	uu := picker.pick()
	if u == uu {
		t.Errorf("url picked = %+v, want other possible urls", uu)
	}
}

func mustNewURLPicker(t *testing.T, us []string) *urlPicker {
	urls := testutil.MustNewURLs(t, us)
	return newURLPicker(urls)
}
