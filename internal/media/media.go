// Package media implements private staged downloads and conservative local matching.
package media

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var (
	// ErrUnsafe indicates an unsafe filesystem path or object.
	ErrUnsafe = errors.New("unsafe destination")
	// ErrTransfer indicates a failed or incomplete transfer; it contains no remote URL.
	ErrTransfer = errors.New("media transfer failed")
)

// ClientDoer is the HTTP boundary used for media requests.
type ClientDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Name returns a portable leaf name with a stable identity suffix to prevent collisions.
func Name(original, identity string) string {
	var b strings.Builder
	for _, r := range original {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(".-_", r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	s := strings.Trim(b.String(), ". ")
	if len(s) > 140 {
		s = "original"
	}
	if s == "" {
		s = "original"
	}
	ext := filepath.Ext(s)
	sum := sha256.Sum256([]byte(identity))
	return fmt.Sprintf("%s-%x%s", strings.TrimSuffix(s, ext), sum[:12], ext)
}

// Directory creates missing directories privately and rejects symlinks in every component.
func Directory(path string, create bool) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return ErrUnsafe
	}
	current := filepath.VolumeName(absolute) + string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(absolute, current), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) && create {
			err = os.Mkdir(current, 0700)
			if err == nil {
				info, err = os.Lstat(current)
			}
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return ErrUnsafe
		}
	}
	return nil
}

// Decision applies filename/size matching without claiming content verification.
func Decision(path string, size int64, overwrite bool) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "downloaded", nil
	}
	if err != nil {
		return "", ErrUnsafe
	}
	if !info.Mode().IsRegular() {
		return "", ErrUnsafe
	}
	if size >= 0 && info.Size() == size {
		return "skipped-name-size-unverified", nil
	}
	if !overwrite {
		return "conflict", nil
	}
	return "replaced", nil
}

// Copy fetches a fresh response into a unique partial file and publishes only a valid copy.
// The injected client must enforce approved HTTPS authorities and credential isolation.
func Copy(ctx context.Context, client ClientDoer, rawURL, path string, size int64, overwrite bool) (string, error) {
	if err := Directory(filepath.Dir(path), true); err != nil {
		return "failed", err
	}
	outcome, err := Decision(path, size, overwrite)
	if err != nil {
		return "failed", err
	}
	if outcome == "conflict" || outcome == "skipped-name-size-unverified" {
		return outcome, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "failed", ErrTransfer
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "failed", ctx.Err()
		}
		return "failed", ErrTransfer
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "failed", ErrTransfer
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".icloud-*.part")
	if err != nil {
		return "failed", ErrUnsafe
	}
	defer os.Remove(f.Name())
	n, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if ctx.Err() != nil {
		return "failed", ctx.Err()
	}
	if copyErr != nil || closeErr != nil || (size >= 0 && n != size) || (resp.ContentLength >= 0 && n != resp.ContentLength) {
		return "failed", ErrTransfer
	}
	// Recheck before publication; concurrent invocations remain unsupported.
	latest, err := Decision(path, size, overwrite)
	if err != nil {
		return "failed", err
	}
	if latest == "conflict" || latest == "skipped-name-size-unverified" {
		return latest, nil
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return "failed", ErrUnsafe
	}
	return outcome, nil
}
