//go:build !linux && !darwin

package state

import "os"

// owned fails closed until platform-specific private storage is supported.
func owned(os.FileInfo) bool { return false }
