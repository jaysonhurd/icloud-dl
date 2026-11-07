package state

import (
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
