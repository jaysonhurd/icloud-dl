package state

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestReportRoundTrip verifies advisory persistence and preserving unknown data.
func TestReportRoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ACL persistence intentionally unsupported")
	}
	root := t.TempDir()
	report := Report{Version: 1, Items: []Item{{ID: "synthetic", Path: "originals/file", Outcome: "downloaded"}}}
	if err := Save(root, report); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(root)
	if err != nil || len(loaded.Items) != 1 {
		t.Fatal(loaded, err)
	}
	path := filepath.Join(root, ".icloud-dl", "manifest.json")
	unknown := []byte(`{"version":999}`)
	if err := os.WriteFile(path, unknown, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Save(root, report); err == nil {
		t.Fatal("overwrote unknown report")
	}
	data, _ := os.ReadFile(path)
	if string(data) != string(unknown) {
		t.Fatal("unknown report changed")
	}
}

// TestReportJSONCompatibility retains legacy report readability and nil-list encoding.
func TestReportJSONCompatibility(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ACL persistence intentionally unsupported")
	}
	root := t.TempDir()
	if err := Save(root, Report{Version: 1}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".icloud-dl", "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Contains(data, []byte(`"items": null`)) {
		t.Fatalf("nil-list wire representation changed: %v", err)
	}
	legacy := []byte(`{"version":1,"updated":"2026-11-07T00:00:00Z","listing_complete":false,"items":null}`)
	if err := os.WriteFile(path, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err != nil {
		t.Fatalf("legacy report unreadable: %v", err)
	}
	// V2 rejects duplicate object members rather than accepting ambiguous report data.
	duplicate := []byte(`{"version":1,"version":1,"items":[]}`)
	if err := os.WriteFile(path, duplicate, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("accepted duplicate member")
	}
	if err := Save(root, Report{Version: 1}); err == nil {
		t.Fatal("overwrote ambiguous report")
	}
	preserved, _ := os.ReadFile(path)
	if !bytes.Equal(preserved, duplicate) {
		t.Fatal("ambiguous data changed")
	}
}
