// Package state stores advisory progress without authentication secrets.
package state

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jaysonhurd/icloud-dl/internal/media"
)

var (
	// ErrReport indicates missing, unreadable, unknown, or unsafe report data.
	ErrReport = errors.New("report unavailable or unsafe; existing data preserved")
	// ErrPrivacy indicates that private storage cannot be established.
	ErrPrivacy = errors.New("private storage unavailable")
)

// Item contains only necessary private mapping and outcome metadata.
type Item struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Outcome string `json:"outcome"`
}

// Report is an advisory snapshot, never proof of a complete backup.
type Report struct {
	Version         int       `json:"version"`
	Updated         time.Time `json:"updated"`
	Limited         bool      `json:"limited,omitempty"`
	ListingComplete bool      `json:"listing_complete"`
	Items           []Item    `json:"items"`
}

// Private checks restrictive permissions; Windows persistence awaits ACL support.
func Private(path string, directory bool) error {
	if runtime.GOOS == "windows" {
		return ErrPrivacy
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 || info.IsDir() != directory || !owned(info) {
		return ErrPrivacy
	}
	if !directory && !info.Mode().IsRegular() {
		return ErrPrivacy
	}
	return nil
}

// Load reads a bounded, private report without modifying any files.
func Load(root string) (Report, error) {
	var report Report
	dir := filepath.Join(root, ".icloud-dl")
	path := filepath.Join(dir, "manifest.json")
	if media.Directory(dir, false) != nil || Private(dir, true) != nil || Private(path, false) != nil {
		return report, ErrReport
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() > 16<<20 {
		return report, ErrReport
	}
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &report) != nil || report.Version != 1 {
		return Report{}, ErrReport
	}
	return report, nil
}

// Save replaces a recognized report; unknown existing data is never overwritten.
func Save(root string, report Report) error {
	dir := filepath.Join(root, ".icloud-dl")
	path := filepath.Join(dir, "manifest.json")
	if _, err := os.Lstat(path); err == nil {
		if _, err = Load(root); err != nil {
			return ErrReport
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return ErrReport
	}
	if media.Directory(dir, true) != nil || Private(dir, true) != nil {
		return ErrPrivacy
	}
	report.Version = 1
	report.Updated = time.Now().UTC()
	data, err := json.Marshal(report, jsontext.WithIndent("  "), json.FormatNilSliceAsNull(true))
	if err != nil {
		return ErrReport
	}
	f, err := os.CreateTemp(dir, ".report-*")
	if err != nil {
		return ErrReport
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		return ErrReport
	}
	if os.Rename(f.Name(), path) != nil {
		return ErrReport
	}
	return nil
}
