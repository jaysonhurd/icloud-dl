//go:build !linux

package main

import (
	"context"
	"errors"
)

var (
	// ErrTerminal indicates an unsupported native terminal implementation.
	ErrTerminal = errors.New("native no-echo terminal input is currently supported only on Linux")
)

// terminal is an unavailable-input boundary on unsupported native platforms.
type terminal struct{}

// openTerminal refuses credential prompts until native no-echo support exists.
func openTerminal() (*terminal, error) { return nil, ErrTerminal }

// Close is a no-op on unsupported systems.
func (*terminal) Close() error { return nil }

// Prompt refuses unsafe input rather than echoing credentials.
func (*terminal) Prompt(context.Context, string, bool) (string, error) { return "", ErrTerminal }
