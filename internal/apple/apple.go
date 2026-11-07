// Package apple defines a fail-closed boundary for the unsupported Apple protocol.
// Independent SRP authentication and production endpoint admission are not yet validated.
package apple

import (
	"context"
	"errors"

	"github.com/jaysonhurd/icloud-dl/internal/archive"
)

var (
	// ErrNotReady prevents unvalidated production authentication or misleading empty success.
	ErrNotReady = errors.New("Apple protocol adapter is not implemented: SRP, verification and endpoint admission require validation")
)

// Source is the production adapter placeholder; it makes no network requests.
type Source struct{}

// List explicitly fails until authenticated enumeration is implemented.
func (Source) List(context.Context) ([]archive.Asset, error) { return nil, ErrNotReady }

// URL explicitly fails rather than inventing a media endpoint.
func (Source) URL(context.Context, archive.Asset) (string, error) { return "", ErrNotReady }
