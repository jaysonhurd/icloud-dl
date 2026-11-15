//go:build linux

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

var (
	// ErrTerminal indicates unavailable terminal input without exposing account information.
	ErrTerminal = errors.New("a local interactive terminal is required")
)

// terminal owns the local controlling terminal and does not accept redirected credentials.
type terminal struct {
	file   *os.File
	reader *bufio.Reader
}

// openTerminal opens a controlling TTY; credentials cannot come from flags or environment.
func openTerminal() (*terminal, error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, ErrTerminal
	}
	return &terminal{file: f, reader: bufio.NewReader(f)}, nil
}

// Close releases the terminal descriptor.
func (t *terminal) Close() error { return t.file.Close() }

// Prompt disables terminal echo for secrets and restores it even after cancellation.
func (t *terminal) Prompt(ctx context.Context, label string, secret bool) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	var original syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, t.file.Fd(), syscall.TCGETS, uintptr(unsafe.Pointer(&original))); errno != 0 {
		return "", ErrTerminal
	}
	if secret {
		hidden := original
		hidden.Lflag &^= syscall.ECHO
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, t.file.Fd(), syscall.TCSETS, uintptr(unsafe.Pointer(&hidden))); errno != 0 {
			return "", ErrTerminal
		}
		defer func() {
			syscall.Syscall(syscall.SYS_IOCTL, t.file.Fd(), syscall.TCSETS, uintptr(unsafe.Pointer(&original)))
			fmt.Fprintln(t.file)
		}()
	}
	if _, err := fmt.Fprint(t.file, label); err != nil {
		return "", ErrTerminal
	}
	type answer struct {
		value string
		err   error
	}
	done := make(chan answer, 1)
	go func() {
		line, err := t.reader.ReadString('\n')
		done <- answer{strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result := <-done:
		if result.err != nil {
			return "", ErrTerminal
		}
		return result.value, nil
	}
}
