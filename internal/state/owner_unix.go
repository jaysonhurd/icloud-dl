//go:build linux || darwin

package state

import (
	"os"
	"syscall"
)

// owned requires filesystem ownership by the current effective user.
func owned(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}
