package main

import (
	"bytes"
	"context"
	"testing"
)

// TestOffline verifies safe commands and invalid input without runtime writes.
func TestOffline(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code int
	}{

		{[]string{"download", "--destination", t.TempDir(), "--limit", "-1"}, 2},
		{[]string{"download", "--destination", t.TempDir(), "--limit", "abc"}, 2},
		{[]string{"status", "--destination", t.TempDir(), "--limit", "1"}, 2},
		{[]string{"--help"}, 0}, {[]string{"--version"}, 0}, {[]string{"invalid"}, 2}, {[]string{"download"}, 2}, {[]string{"login", "--profile", "../bad"}, 2},
	} {
		var out, stderr bytes.Buffer
		if got := run(context.Background(), tc.args, &out, &stderr); got != tc.code {
			t.Fatalf("%v: %d, want %d", tc.args, got, tc.code)
		}
	}
}
