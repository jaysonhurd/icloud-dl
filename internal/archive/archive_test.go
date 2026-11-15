package archive

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type source struct{}

// List supplies a synthetic partial listing with a duplicate logical asset.
func (source) List(context.Context) ([]Asset, error) {
	return []Asset{{ID: "one", Name: "file", Supported: true}, {ID: "one", Name: "file", Supported: true}, {ID: "two", Supported: false}}, errors.New("page failed")
}

// URL must never be called by preview.
func (source) URL(context.Context, Asset) (string, error) { panic("preview requested media") }

// TestPartialDryRun verifies usable partial entries and write-free duplicate handling.
func TestPartialDryRun(t *testing.T) {
	report, err := Run(context.Background(), source{}, nil, Config{Destination: t.TempDir(), DryRun: true})
	if err == nil || report.ListingComplete || len(report.Items) != 2 || report.Items[0].Outcome != "planned" || report.Items[1].Outcome != "unsupported" {
		t.Fatal(report, err)
	}
}

type limitedSource struct {
	lookups int
	fail    bool
}

// List supplies distinct supported originals, a duplicate, and an unsupported item.
func (s *limitedSource) List(context.Context) ([]Asset, error) {
	return []Asset{{ID: "unsupported"}, {ID: "a", Name: "a", Supported: true}, {ID: "a", Name: "a", Supported: true}, {ID: "b", Name: "b", Supported: true}, {ID: "c", Name: "c", Supported: true}}, nil
}

// URL counts selected files even when fresh lookup fails.
func (s *limitedSource) URL(context.Context, Asset) (string, error) {
	s.lookups++
	if s.fail {
		return "", errors.New("synthetic lookup failure")
	}
	return "https://synthetic.invalid/media", nil
}

type countingClient struct{ calls int }

// Do returns one synthetic original without contacting a network.
func (c *countingClient) Do(*http.Request) (*http.Response, error) {
	c.calls++
	return &http.Response{StatusCode: 200, ContentLength: 0, Body: io.NopCloser(strings.NewReader(""))}, nil
}

// TestLimit bounds transfers, failed lookups, and previews without counting duplicates.
func TestLimit(t *testing.T) {
	for _, tc := range []struct {
		limit     int
		dry, fail bool
		want      int
		limited   bool
	}{
		{1, false, false, 1, true}, {2, false, false, 2, true}, {0, false, false, 3, false}, {5, false, false, 3, false},
		{1, false, true, 1, true}, {1, true, false, 0, true},
	} {
		src := &limitedSource{fail: tc.fail}
		client := &countingClient{}
		report, _ := Run(context.Background(), src, client, Config{Destination: t.TempDir(), Limit: tc.limit, DryRun: tc.dry})
		if src.lookups != tc.want || report.Limited != tc.limited {
			t.Fatalf("limit=%d dry=%t: lookups=%d limited=%t", tc.limit, tc.dry, src.lookups, report.Limited)
		}
		if !tc.fail && client.calls != tc.want {
			t.Fatalf("media calls=%d, want %d", client.calls, tc.want)
		}
	}
}

// TestInvalidLimit rejects invalid configuration before catalog access.
func TestInvalidLimit(t *testing.T) {
	_, err := Run(context.Background(), nil, nil, Config{Limit: -1})
	if !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
}
