package apple

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

type doer func(*http.Request) (*http.Response, error)

// Do implements the test transport.
func (d doer) Do(r *http.Request) (*http.Response, error) { return d(r) }

// TestGuard exercises prohibited egress and media credential isolation.
func TestGuard(t *testing.T) {
	calls := 0
	g := Guard{Hosts: map[string]bool{"synthetic.invalid": true}, Media: true, Transport: doer(func(r *http.Request) (*http.Response, error) {
		calls++
		if len(r.Header) != 0 {
			t.Fatal("forwarded credentials")
		}
		return nil, errors.New("SECRET signed-url")
	})}
	for _, raw := range []string{"http://synthetic.invalid", "https://synthetic.invalid.evil", "https://user:pass@synthetic.invalid", "https://synthetic.invalid:444"} {
		req, _ := http.NewRequest("GET", raw, nil)
		if _, err := g.Do(req); !errors.Is(err, ErrEndpoint) {
			t.Fatal("accepted endpoint")
		}
	}
	if calls != 0 {
		t.Fatal("prohibited egress")
	}
	req, _ := http.NewRequest("GET", "https://synthetic.invalid", nil)
	req.Header.Set("Authorization", "SECRET")
	_, err := g.Do(req)
	if err == nil || strings.Contains(err.Error(), "SECRET") {
		t.Fatal("unsafe transport error")
	}
	if calls != 1 {
		t.Fatal("allowed request not sent")
	}
}
