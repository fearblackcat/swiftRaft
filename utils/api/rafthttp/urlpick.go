package rafthttp

import (
	"net/url"
	"sync"

	"github.com/fearblackcat/swiftRaft/utils/pkg/types"
)

type urlPicker struct {
	mu     sync.Mutex // guards urls and picked
	urls   types.URLs
	picked int
}

func newURLPicker(urls types.URLs) *urlPicker {
	return &urlPicker{
		urls: urls,
	}
}

func (p *urlPicker) update(urls types.URLs) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.urls = urls
	p.picked = 0
}

func (p *urlPicker) pick() url.URL {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.urls[p.picked]
}

// unreachable notices the picker that the given url is unreachable,
// and it should use other possible urls.
func (p *urlPicker) unreachable(u url.URL) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if u == p.urls[p.picked] {
		p.picked = (p.picked + 1) % len(p.urls)
	}
}
