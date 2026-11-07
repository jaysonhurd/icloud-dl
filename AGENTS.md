# icloud-dl project instructions

## Project overview and scope

This repository contains an initial Go implementation and a design kit for a personal Go
CLI that downloads the owner's iCloud Photos originals directly to Debian/Ubuntu Linux
amd64, Windows amd64, and macOS arm64/amd64. The required toolchain is
**Go 1.27.3**, module `github.com/jaysonhurd/icloud-dl`, entry point `cmd/icloud-dl`.
The offline media/report core exists; Apple authentication/catalog and the wizard
remain pending. See README for actual behavior and verification commands. This is a best-effort personal
utility: prefer ordinary temporary downloads, filename/size reruns, bounded retries
and useful summaries. Do not add transaction journals, transfer receipts, per-item
durable checkpoints, multi-process guarantees or exhaustive reliability gates.

Read [README.md](README.md), the
[constitution](.specify/memory/constitution.md), and the feature's
[specification](specs/001-icloud-photos-download/spec.md),
[research](specs/001-icloud-photos-download/research.md),
[plan](specs/001-icloud-photos-download/plan.md), and
[tasks](specs/001-icloud-photos-download/tasks.md) before changing the design.
Preserve project-specific requirements over generic preferences.

## Architecture and development workflow

- The plan proposes `cmd/icloud-dl` and `internal/archive`, `internal/apple`,
  `internal/state`, and `internal/media`. These packages now exist; the Apple adapter fails explicitly until implemented. Keep
  later implementation within the documented responsibilities; justify new layers.
- Apple authentication and Photos services are external systems with private,
  unsupported protocols. Keep protocol logic behind narrow consumer interfaces.
  Label source observations, inferences, and unverified questions separately.
- Follow the README's Spec Kit workflow and preserve stable requirement/task IDs.
  Keep specifications, plans, and tasks consistent as scope changes. Preserve
  generated templates, helpers, and skills; do not refresh tooling implicitly.
- Current tooling prerequisites are Python 3.11+, Git, and Bash for checks;
  Specify CLI 1.3.2, uv, and Codex support the documented feature workflow.
- Build and check the existing module with Go 1.27.3 using README commands.
  No live integration-test environment or Apple account configuration exists.
- Preserve pre-existing work. Keep changes scoped and straightforward, explain
  significant tradeoffs/dependencies, and proceed autonomously with routine,
  reversible work. Do not stage, commit, publish, merge, or change remote settings
  unless the user's request authorizes that scope.

## Security and compatibility requirements

- Remote access is read-only: authenticate, enumerate, resolve originals, and
  download. No upload, delete, edit, rename, move, cloud relay, or two-way sync.
  Preserve the declared first-release scope and unsupported-format reporting.
- No-argument invocation is a local wizard: username, no-echo password, destination,
  overwrite [y/N] (default N). Compare mapped filename and size; label same-size
  skips unverified. Different-size files require explicit Y and a successful staged copy before
  ordinary platform replacement. This never authorizes remote changes or bypasses security checks.
- Never request passwords/codes in agent chat or put credentials/account data in
  source, fixtures, arguments, configuration, logs, or documentation. Normal Apple
  verification and explicit consent for trust/session persistence are mandatory.
- Future stored session state is owned 0700 directories/0600 files on Unix or
  equivalent restricted current-user ACLs on Windows, separate from
  the protected manifest. Logout is local clearing, not server-side revocation.
- No telemetry, analytics, or remote tracing/exporters. Never log signed URLs,
  cookies, tokens, headers, raw payloads, filenames, email or raw asset IDs. Use
  allowlisted operational fields and safe errors, including in debug mode.
- Preserve original bytes, safe filename mapping, private temporary copies, explicit
  per-run overwrite and honest incomplete outcomes. Interrupted items restart from
  zero; stale partials may need manual removal. The report is advisory and may lag;
  it must not gate otherwise safe copying. Do not weaken authentication or path
  safety to make a transfer work. Run one invocation per profile/destination.
- Routine tests use synthetic fixtures and local doubles. Real-account checks
  require separate explicit opt-in; unresolved protocol gates remain unresolved
  until evidence exists. Upstream icloudpd/rclone code is reference material only,
  not an import, vendored dependency, subprocess, or substantial code-copy source.
- Preserve existing ignore rules for personal agent state, secrets, IDE settings
  and generated logs. Shared Spec Kit JSON metadata must remain available for
  version control; do not hide it with a blanket ignore rule. Create logs only during runtime commands that require them; keep dependencies justified.

## Go conventions for future implementation

- Use idiomatic Go and relevant Effective Go guidance compatible with the chosen
  toolchain. Prefer the standard library, clear names, focused responsibilities,
  small consumer interfaces, composition and explicit dependency injection. Apply
  SOLID pragmatically; avoid abstractions without a concrete purpose.
- Use blank lines between logical sections and grouped `const`/`var` declarations
  at file/function scope where appropriate. Document exported declarations and
  private functions/methods; use JSON tags while preserving intentional wire
  contracts and excluded fields.
- Prefer native `testing` and meaningful success/error cases. Aim for useful coverage and report measured gaps; the personal 95% preference
  is aspirational, not a release gate or a reason for exhaustive crash tests. For HTTP clients inject a ClientDoer
  with `Do(*http.Request) (*http.Response, error)`; preserve suitable handler tests.
- Prefer `net/http` and `log/slog`, fixed message constants and typed structured
  attributes. Default file logging is `logs/icloud-dl.log` beneath the launch working directory; create a missing
  logs directory privately and fail clearly if unsafe/unwritable, with ignored output;
  dry-run must never create a log. Redact errors before logging them.
- Use documented `Err`-prefixed sentinel errors where identity is useful; preserve
  wrapping/caller contracts, use `errors.Is`/`errors.As`, and avoid string matching
  or duplicate logs at every layer. Reserve panic for unrecoverable programmer
  errors. Use `fmt.Errorf` wrapping where needed, never formatted secret-bearing
  logging as a substitute for structured attributes.
- Accept and propagate `context.Context` for blocking/I/O work, honor cancellation
  and deadlines, and release derived contexts. Do not store contexts in structs
  or replace caller contexts with background contexts.
- Validate configuration at startup and inject it explicitly. Account passwords
  and verification codes remain local terminal input, never environment config.
- Evaluate `encoding/json/v2` only against the selected Go version and wire
  compatibility. Never silently upgrade the toolchain or enable experiments.
  Observability must follow the project's local-only policy; do not introduce
  OpenTelemetry dependencies/exporters during preference or documentation updates.

## Formatting and repository validation

Run the existing local/CI commands from the repository root after instruction,
documentation, validation, or workflow changes, unless the user explicitly limits
the task to static checks:

```bash
python3 -m unittest discover -s scripts -p 'test_*.py'
python3 scripts/check_repository.py
```

The tests exercise the validator; the checker covers basic skill metadata,
inline local Markdown links/anchors, and Bash syntax. It does not verify live
protocol behavior, every Markdown/YAML construct, or remote enforcement. Check
requirement/task references and design consistency separately for feature edits.
For static-only tasks, report which checks ran and that tests were not run.

Once Go code is authorized, use Go 1.27.3 and record `go 1.27.3` in go.mod; format changed
files with `gofmt`, and extend existing local/CI tooling with `go build ./...`,
`go test ./...`, and `go vet ./...` using actual module roots/build tags. Use the existing module for these checks. Test meaningful failure cases when
changing validation scripts; avoid duplicate validation systems.

The stable `Repository validation` check targets PRs to and pushes to `main`.
Preserve read-only permissions, timeout, reviewed SHA-pinned Actions, and checkout
without persisted credentials, plus IDE ignores under `.github`. Remote required
checks, security settings and allowed Actions require explicit remote-setup scope
and read-back verification. Preserve stronger protections, selected licenses and
upstream notices; never bypass checks to merge.

Report changed files, actual checks/results, skipped concerns and unresolved
limitations concisely. Never claim an implementation task, live compatibility,
CI run, or remote setting is complete without evidence.
