package apple

import (
	"errors"
	"net/http"
	"net/url"
)

var (
	// ErrEndpoint rejects requests outside a reviewed exact-host HTTPS boundary.
	ErrEndpoint = errors.New("endpoint rejected")
	// ErrHTTP conceals secret-bearing transport errors and response details.
	ErrHTTP = errors.New("Apple request failed")
)

// ClientDoer allows deterministic synthetic HTTP tests.
type ClientDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Guard admits only explicitly reviewed authorities; use a separate instance for media.
// Transport must not follow redirects; RedirectPolicy is provided for http.Client.
type Guard struct {
	Transport ClientDoer
	Hosts     map[string]bool
	Media     bool
}

// Admit requires TLS, the default port, no userinfo, and an exact admitted hostname.
func (g Guard) Admit(u *url.URL) bool {
	return u != nil && u.Scheme == "https" && u.User == nil && (u.Port() == "" || u.Port() == "443") && g.Hosts[u.Hostname()]
}

// Do validates the boundary and strips all caller headers and cookies for media.
func (g Guard) Do(req *http.Request) (*http.Response, error) {
	if req == nil || !g.Admit(req.URL) || g.Transport == nil {
		return nil, ErrEndpoint
	}
	if g.Media && (req.Method != http.MethodGet || req.Body != nil) {
		return nil, ErrEndpoint
	}
	clone := req.Clone(req.Context())
	if g.Media {
		clone.Header = make(http.Header)
	}
	resp, err := g.Transport.Do(clone)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return nil, ErrHTTP
	}
	if resp == nil || resp.Body == nil {
		return nil, ErrHTTP
	}
	return resp, nil
}

// RedirectPolicy refuses redirects; a new reviewed request is required instead.
func RedirectPolicy(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
