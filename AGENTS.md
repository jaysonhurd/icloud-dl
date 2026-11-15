# icloud-dl project instructions

## Project and references

Personal, best-effort, read-only iCloud Photos originals downloader. Module:
`github.com/jaysonhurd/icloud-dl`; entry point: `cmd/icloud-dl`. Use exactly
Go **1.27.3**, `GOTOOLCHAIN=local`, and the explicitly authorized
`GOEXPERIMENT=jsonv2`. No external Go dependencies or live-account test environment.

Read [README](README.md), the [constitution](.specify/memory/constitution.md), and
feature [spec](specs/001-icloud-photos-download/spec.md),
[research](specs/001-icloud-photos-download/research.md),
[plan](specs/001-icloud-photos-download/plan.md), and
[tasks](specs/001-icloud-photos-download/tasks.md) before design changes.
The constitution owns durable security, Go, testing and tooling principles;
project requirements override reusable preferences.

## Implementation boundaries

- `cmd/icloud-dl`: commands, local no-echo terminal input, cancellation, safe output.
- `internal/archive`: policy, bounded selection (`--limit`), outcomes.
- `internal/apple`: independent private-protocol authentication, listing, lookup.
- `internal/media`: safe filenames, filename/size matching, temporary publication.
- `internal/state`: private advisory reports; session persistence is unfinished.

Apple compatibility remains experimental. The owner has reported login-stage
progress and failures; no successful end-to-end library download is established.
Only Linux has no-echo prompts; macOS/Windows runtime support is incomplete.
The wizard, persisted sessions, retry/worker scheduling and full acceptance work
remain pending. Do not mark tasks complete without evidence or add recovery
journals, receipts, durable checkpoints or multi-process guarantees.

Never request account passwords/codes in chat. Routine tests use synthetic data
and injected HTTP clients; live account checks require separate explicit opt-in.
No telemetry/exporters. No-argument wizard is intended behavior, not implemented
behavior. Preserve default-N overwrite, unverified filename/size skips, write-free
dry-run/help/version/status and runtime-only private `CWD/logs/icloud-dl.log`.

## Commands and validation

From the repository root, with Go 1.27.3 on PATH:

```bash
export GOTOOLCHAIN=local
export GOEXPERIMENT=jsonv2
test "$(go env GOVERSION)" = go1.27.3
test -z "$(gofmt -l cmd internal)"
go build ./...
go test -cover ./...
go vet ./...
python3 -m unittest discover -s scripts -p 'test_*.py'
python3 scripts/check_repository.py
```

Format changed Go files explicitly with `gofmt -w`. Build a local binary with
`go build -o ./icloud-dl ./cmd/icloud-dl`; inspect `./icloud-dl --help` offline.
See README for command prerequisites, coverage gaps and validation limits.
CI's `Repository validation` targets main pushes/PRs; tagged builds target
linux/amd64, windows/amd64, darwin/arm64 and darwin/amd64. Compilation does not
prove native behavior, live Apple compatibility, CI execution or remote enforcement.

## Guidance and work preservation

Keep root instructions concise; durable principles belong in the constitution,
feature behavior/design/tasks in `specs/`. Preserve generated Spec Kit skills,
helpers, templates and existing Codex integration. Keep custom skills separate;
do not refresh tooling, install preferences or change integrations implicitly.
The local preference source is `.agents/coding-preferences`; its presence does
not authorize editing its installation or repairing pre-existing skill moves.

Preserve unrelated work, shared JSON, licenses, IDE/secret/log ignores and private
agent state. Do not stage, commit, publish, merge or change remote settings without
explicit scope. Report actual changes/checks, skipped concerns and unresolved
limitations; never claim unperformed checks passed.
