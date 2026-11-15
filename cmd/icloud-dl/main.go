// Command icloud-dl is the entry point for the personal archive utility.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/jaysonhurd/icloud-dl/internal/apple"
	"github.com/jaysonhurd/icloud-dl/internal/archive"
	"github.com/jaysonhurd/icloud-dl/internal/media"
	"github.com/jaysonhurd/icloud-dl/internal/state"
	"strings"
)

const (
	version    = "0.1.0-dev"
	logStarted = "Command started"
)

// main owns signal handling and process exit.
func main() {
	os.Exit(mainExitCode())
}

// mainExitCode releases signal resources before main terminates the process.
func mainExitCode() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	return run(ctx, os.Args[1:], os.Stdout, os.Stderr)
}

// run parses commands without accepting credentials on the command line.
func run(ctx context.Context, args []string, out, stderr io.Writer) int {
	if ctx.Err() != nil {
		return 130
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Use: icloud-dl download --destination PATH --limit N. No-argument wizard is still pending.")
		return 4
	}
	switch args[0] {
	case "--version":
		fmt.Fprintln(out, "icloud-dl", version)
		return 0
	case "--help", "help":
		fmt.Fprintln(out, "icloud-dl: personal best-effort originals downloader (initial build)\nCommands: login, download, status, logout\nDownload flags: --destination PATH --profile NAME --workers 1..4 --limit N --overwrite --dry-run\nLimit: maximum selected files; 0 means unlimited.\nMedia layout: DESTINATION/originals. Same name/size skips are unverified.\nExperimental private Apple protocol; credentials are prompted locally. Linux terminal support only.")
		return 0
	}
	command := args[0]
	if command != "login" && command != "download" && command != "status" && command != "logout" {
		fmt.Fprintln(stderr, "Unknown command; use --help.")
		return 2
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	destination := flags.String("destination", "", "archive root")
	profile := flags.String("profile", "default", "local profile")
	workers := flags.Int("workers", 2, "workers 1..4")
	var limit int
	if command == "download" {
		flags.IntVar(&limit, "limit", 0, "maximum selected files; 0 means unlimited")
	}
	overwrite := flags.Bool("overwrite", false, "permit local replacement")
	dry := flags.Bool("dry-run", false, "write-free preview")
	remember := flags.Bool("remember-session", false, "opt-in session storage")
	if flags.Parse(args[1:]) != nil || flags.NArg() != 0 || *workers < 1 || *workers > 4 || limit < 0 || !validProfile(*profile) {
		fmt.Fprintln(stderr, "Invalid options; use --help.")
		return 2
	}
	if (command == "status" || command == "download") && *destination == "" {
		fmt.Fprintln(stderr, "--destination is required.")
		return 2
	}
	if command == "status" {
		report, err := state.Load(*destination)
		if err != nil {
			fmt.Fprintln(out, "No usable private report; no files changed.")
			return 3
		}
		fmt.Fprintf(out, "Last saved report (may be stale): %s; observed items: %d; listing complete: %t; limited selection: %t\n", report.Updated.Format("2006-01-02T15:04:05Z"), len(report.Items), report.ListingComplete, report.Limited)
		return 0
	}
	if !*dry {
		log, err := openLog()
		if err != nil {
			fmt.Fprintln(stderr, "Cannot open private CWD logs/icloud-dl.log.")
			return 5
		}
		slog.New(slog.NewJSONHandler(log, nil)).Info(logStarted, slog.String("command", command))
		if log.Close() != nil {
			return 5
		}
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return 130
	}
	if *remember {
		fmt.Fprintln(stderr, "Session persistence is not implemented; omit --remember-session for an ephemeral login.")
		return 2
	}
	if command == "logout" {
		fmt.Fprintln(out, "This build stores no session credentials. Local logout does not revoke Apple sessions.")
		return 0
	}
	tty, err := openTerminal()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	defer tty.Close()
	account, err := tty.Prompt(ctx, "Apple account username: ", false)
	if err != nil {
		return promptError(ctx, stderr)
	}
	password, err := tty.Prompt(ctx, "Apple account password (not saved): ", true)
	if err != nil {
		return promptError(ctx, stderr)
	}
	if strings.TrimSpace(account) == "" || password == "" {
		fmt.Fprintln(stderr, "Username and password are required.")
		return 4
	}
	source := apple.New()
	fmt.Fprintln(out, "Experimental read-only Apple Photos access. Hidden/deleted/shared libraries and Live Photo companions are excluded.")
	if err := source.Login(ctx, strings.TrimSpace(account), password, tty.Prompt, !*dry); err != nil {
		password = ""
		if ctx.Err() != nil {
			return 130
		}
		fmt.Fprintln(stderr, err)
		return 4
	}
	password = ""
	if command == "login" {
		fmt.Fprintln(out, "Authenticated for this invocation; no reusable credentials were saved.")
		return 0
	}
	report, err := archive.Run(ctx, source, apple.NewMediaClient(), archive.Config{Destination: *destination, Overwrite: *overwrite, DryRun: *dry, Limit: limit})
	fmt.Fprintln(out, apple.Summary(len(report.Items), report.Limited))
	counts := map[string]int{}
	for _, item := range report.Items {
		counts[item.Outcome]++
	}
	fmt.Fprintf(out, "Downloaded: %d; replaced: %d; unverified skips: %d; failed: %d; conflicts: %d; unsupported: %d; planned: %d\n", counts["downloaded"], counts["replaced"], counts["skipped-name-size-unverified"], counts["failed"], counts["conflict"], counts["unsupported"], counts["planned"])
	if ctx.Err() != nil {
		return 130
	}
	if !report.ListingComplete {
		fmt.Fprintln(stderr, "Listing incomplete; known usable downloads may still have succeeded.")
	}
	if err != nil {
		fmt.Fprintln(stderr, "Run incomplete or advisory report could not be saved. Existing successful copies remain available.")
		return 3
	}
	return 0
}

// validProfile accepts only bounded portable profile identifiers.
func validProfile(s string) bool {
	if len(s) < 1 || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// openLog creates private CWD logging and refuses unsafe existing objects.
func openLog() (*os.File, error) {
	dir := "logs"
	path := filepath.Join(dir, "icloud-dl.log")
	if media.Directory(dir, true) != nil || state.Private(dir, true) != nil {
		return nil, state.ErrPrivacy
	}
	if _, err := os.Lstat(path); err == nil {
		if state.Private(path, false) != nil {
			return nil, state.ErrPrivacy
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, state.ErrPrivacy
	}
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
}

// promptError renders only a safe input category and preserves cancellation exit semantics.
func promptError(ctx context.Context, out io.Writer) int {
	if ctx.Err() != nil {
		return 130
	}
	fmt.Fprintln(out, "Terminal input failed; no credentials were saved.")
	return 4
}
