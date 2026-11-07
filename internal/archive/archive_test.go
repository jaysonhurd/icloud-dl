package archive

import (
	"context"
	"errors"
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
