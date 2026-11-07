package media

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type doer func(*http.Request) (*http.Response, error)

// Do implements the injected HTTP seam.
func (f doer) Do(r *http.Request) (*http.Response, error) { return f(r) }

// TestCopyPolicies verifies success, unverified skip, consent, failed replacement and cleanup.
func TestCopyPolicies(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "originals", "photo.jpg")
	body := "abc"
	client := doer(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, ContentLength: int64(len(body)), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	for _, tc := range []struct {
		size      int64
		overwrite bool
		want      string
		fail      bool
	}{
		{3, false, "downloaded", false}, {3, true, "skipped-name-size-unverified", false}, {4, false, "conflict", false}, {4, true, "failed", true},
	} {
		outcome, err := Copy(context.Background(), client, "https://synthetic.invalid/media", path, tc.size, tc.overwrite)
		if outcome != tc.want || (err != nil) != tc.fail {
			t.Fatalf("got %s, %v; want %s", outcome, err, tc.want)
		}
	}
	data, _ := os.ReadFile(path)
	if string(data) != "abc" {
		t.Fatal("failed copy changed final")
	}
	parts, _ := filepath.Glob(filepath.Join(root, "originals", "*.part"))
	if len(parts) != 0 {
		t.Fatal("partial leaked")
	}
	body = "longer"
	outcome, err := Copy(context.Background(), client, "https://synthetic.invalid/media", path, 6, true)
	if err != nil || outcome != "replaced" {
		t.Fatal(outcome, err)
	}
}

// TestUnsafe verifies path confinement and refusing symlink finals.
func TestUnsafe(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside")
	os.WriteFile(outside, []byte("safe"), 0600)
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("symlinks unavailable")
	}
	if _, err := Decision(link, 4, true); err == nil {
		t.Fatal("accepted symlink")
	}
	for _, name := range []string{"../../escape", "CON", "a/b\\c", strings.Repeat("ü", 200), ""} {
		leaf := Name(name, "synthetic-asset")
		if filepath.Base(leaf) != leaf || strings.ContainsAny(leaf, "/\\") || len(leaf) > 200 {
			t.Fatal("unsafe leaf")
		}
	}
	if Name("a.jpg", "one") == Name("a.jpg", "two") {
		t.Fatal("collision")
	}
}

// TestCancellation verifies that cancellation never publishes a completed item.
func TestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := doer(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })
	path := filepath.Join(t.TempDir(), "file")
	if _, err := Copy(ctx, client, "https://synthetic.invalid", path, 1, false); err == nil {
		t.Fatal("cancel succeeded")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("published cancelled copy")
	}
}
