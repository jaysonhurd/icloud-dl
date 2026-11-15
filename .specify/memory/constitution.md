<!--
Sync Impact Report
Version: 3.1.0 -> 3.1.1 (guidance organization and current implementation clarifications).
Root instructions now reference durable principles instead of duplicating them.
Records the owner's existing explicit JSON v2 experiment authorization; no new
experiment, dependency, runtime behavior or privacy obligation is introduced.
README and active plan version references aligned. Generated templates/skills,
existing integration and pre-existing skill relocations preserved.
-->
# icloud-dl Constitution

## Core Principles

### I. Requirements Before Implementation

Each feature MUST define users, observable behavior, and measurable acceptance
criteria before planning implementation. Technical decisions and unresolved
questions MUST be recorded in the plan. Specifications, plans, and tasks MUST
remain consistent as scope changes. Setup MUST NOT invent application features.

### II. Simple, Maintainable Design

Changes MUST stay within agreed scope and established package boundaries. Code
MUST use clear names, focused responsibilities, small consumer interfaces, and
explicit dependency injection. Apply all five SOLID principles pragmatically:

- **Single Responsibility:** packages, types, and functions MUST have focused,
  cohesive responsibilities.
- **Open/Closed:** extend stable behavior through composition or focused
  interfaces where a concrete extension need exists.
- **Liskov Substitution:** implementations MUST preserve interface behavior,
  including errors, cancellation, and side-effect contracts.
- **Interface Segregation:** define small interfaces around consumer needs;
  callers MUST NOT depend on methods they do not use.
- **Dependency Inversion:** inject infrastructure dependencies explicitly;
  high-level archive policy MUST NOT depend on private Apple HTTP/schema details.

New abstractions and production dependencies MUST have a concrete justification;
significant tradeoffs and dependency choices MUST be documented for review. Prefer
the standard library when it meets the requirement, and preserve existing package
boundaries rather than adding layers solely to satisfy a pattern.

### III. Meaningful Verification

Changed behavior MUST be verified against acceptance criteria and meaningful
failure cases. Formatting and relevant checks MUST run after changes, and actual
results, blocked checks, and limitations MUST be reported. New validation scripts
MUST include tests for relevant failures. Tests MUST protect behavior rather than
merely mirror implementation or inflate coverage.

Go tests MUST use the standard `testing` package and native tooling by
default, cover meaningful success and error paths, and aim for at least 95%
statement coverage where practical. Report the measured scope, percentage and
reasons for any shortfall; coverage MUST NOT replace acceptance or failure-path
verification. For this best-effort personal utility the coverage percentage is
aspirational, not a release gate; prioritize useful tests over exhaustive fault
injection. No Go coverage claim applies while there is no Go code.

HTTP client tests MUST inject a small `ClientDoer` interface with
`Do(*http.Request) (*http.Response, error)` rather than requiring a live server.
This preference does not prohibit appropriate handler tests using `httptest`.
Use synthetic fixtures, fake clocks/randomness, and temporary directories for
deterministic tests of retries, cancellation, filesystem faults and privacy.

### IV. Idiomatic Go

Go is the default for backend work unless requirements justify another choice.
The required toolchain MUST be Go 1.27.3 for local development and CI. The
`go.mod` MUST declare `go 1.27.3`; do not silently select a newer toolchain. Go code
MUST follow Effective Go and the following project conventions:

- **Style and documentation:** format changed Go files with `gofmt`. Use blank
  lines between logical sections and prefer grouped `const (...)` and `var (...)`
  declarations at the top of the appropriate file/function scope. Use concise
  package names and idiomatic MixedCaps identifiers. Exported declarations MUST
  have useful Go doc comments; private functions/methods MUST also have concise
  comments. Private variables/constants do not need comments by default.
- **JSON contracts:** struct fields MUST have appropriate JSON tags unless a
  documented exception applies. Preserve intentional wire names, excluded fields,
  and existing caller contracts. Use `encoding/json/v2` with the owner-authorized `GOEXPERIMENT=jsonv2`
  on Go 1.27.3, locally and in CI; preserve existing report wire behavior. Do not
  implicitly upgrade Go or enable additional experiments. Private runtime-only
  structs need not have JSON tags; never serialize credential-bearing structs.
- **HTTP:** prefer standard-library `net/http` and middleware patterns where
  applicable. Keep HTTP clients injectable and retain the separate authentication
  and media trust boundaries defined by the product design.
- **Errors:** stable sentinel errors MUST be package-level, `Err`-prefixed, grouped
  declarations with comments for exported errors. Use typed errors when callers
  need structured detail; preserve error identity and behavior. Handle returned
  errors explicitly, add safe operation context with `fmt.Errorf`/`%w` when useful,
  and inspect chains with `errors.Is`/`errors.As`, not error-string comparisons.
  Do not log the same error at every layer. Reserve panic for unrecoverable
  programmer errors, not account challenges or routine transfer failures.
- **Contexts and cleanup:** blocking/I/O operations MUST accept
  `context.Context` as the first parameter, propagate caller cancellation and
  deadlines, and release derived contexts. Do not store contexts in structs or
  replace caller contexts with `context.Background()`. Close files and HTTP
  response bodies and release locks on all paths, using `defer` where appropriate;
  handle copy/close errors instead of presenting an incomplete download as finished.
- **Configuration:** validate configuration at startup and pass configuration and
  dependencies explicitly. Preserve established flags and environment conventions;
  the password/code terminal-only rule overrides generic environment-secret advice.
- **Logging:** prefer `log/slog`, named constants for most fixed messages, and
  typed structured attributes for changing values. Do not build diagnostic
  messages with formatted or concatenated private data. Apply the privacy
  allowlist before emission; arbitrary wrapped errors are not safe log fields.
  Default file logging MUST use `logs/icloud-dl.log` beneath the launch working
  directory, creating the missing directory with private owner-only access. Fail
  clearly on an unsafe/unwritable location; never silently relocate logs. Keep
  generated output ignored. Help, version and offline status remain write-free. Dry-run MUST NOT create logs or state.

Plans MUST address tracing, metrics, and logging lifecycle before introducing
observability dependencies. The generic OpenTelemetry preference is subordinate
to principle VIII: instrumentation, if later justified, MUST remain local with
no network exporters, collector endpoints, telemetry or analytics. Document new
dependencies and resource cleanup; this amendment adds none.

Effective Go's historical examples MUST be checked against the project's chosen
toolchain before adoption. Preserve this project's explicit contracts and record
intentional overrides. Refresh the local reference only on an explicit source
refresh request. Instruction/preference updates alone MUST NOT introduce application features,
runtime output or unjustified dependencies.

### V. Reproducible Project Tooling

Shared Spec Kit infrastructure, agent skills, and feature documents MUST remain
available for version control. Local validation and CI MUST run consistent
commands for applicable skill metadata, Markdown links and anchors, and shell
syntax. Once Go modules exist, checks MUST also cover formatting, build, tests,
and vet with documented versions and build tags. The README MUST document
commands, prerequisites, and validation limits.

Extend existing validation rather than creating a parallel system. For real Go
modules, local/CI checks MUST reject unformatted project code and run
`go build ./...`, `go test ./...`, and `go vet ./...` from the documented module
roots with supported versions and required tags. CI MUST NOT silently format
and pass. Do not create a module merely to make checks applicable. A pushed-tag build
workflow MUST verify source, compile all four targets using the declared toolchain,
and upload packaged binaries/checksums with read-only permissions, reviewed
SHA-pinned Actions and timeouts. Missing application source MUST fail explicitly;
never upload a placeholder or claim successful compilation from configuration
alone. Publishing a GitHub Release, signing/notarizing or changing remote policy
requires separate scope.

### VI. Read-Only Personal Archive

The product MUST be a Go CLI for the owner's own primary personal iCloud
Photos library, with originals downloaded directly to the chosen local destination.
Build targets are Debian/Ubuntu Linux amd64, Windows amd64, macOS Apple Silicon
arm64 and macOS Intel amd64. Test available platforms pragmatically and label untested builds honestly;
compiling a binary does not prove native runtime behavior. It MUST NOT upload,
delete, edit, rename, move or otherwise mutate remote
content. Authentication, enumeration, lookup and download are the only admitted
remote operations. No Mac or cloud relay is required. Initial scope excludes
hidden/recently deleted items and shared libraries/albums and MUST be disclosed.
It is a repeatable one-time archive, not two-way or continuous synchronization.

The Apple protocol MUST be described as private, unsupported and changeable.
The Go program MUST be independently implemented, without importing, vendoring,
executing, or substantially copying icloudpd/rclone code. Important observations
MUST have precise attributable source/revision evidence; inferences and unresolved
validation MUST be labeled separately. Unknown schema or asset types MUST NOT
silently vanish or be represented as supported originals.

### VII. Authentication Without Weakened Safeguards

Passwords and verification codes MUST be entered only through local no-echo
prompts and MUST NOT be persisted, logged, passed through arguments/configuration,
or requested in agent conversation. Store only necessary cookies/session/trust
state after explicit consent, in owned 0700 directories and 0600 files on Unix
(or equivalent restricted current-user ACLs on Windows), with
symlink/ownership checks and no password-derived material. Reauthentication and
verification MUST preserve normal Apple controls. Device/session trust or code
delivery requires explicit user action; no automatic security-setting changes.
Declined or unsupported challenges MUST stop safely. Local logout MUST clear only
selected credential state and explain that it is not server revocation or secure
erasure. Retries and convenience MUST NEVER weaken these rules.

### VIII. Private Content and Diagnostics

The program MUST make no telemetry/analytics calls or remote tracing exports.
Media/metadata MUST go only between the PC and validated Apple/authenticated
Apple-authorized delivery endpoints needed for the requested operation. Credentials
MUST NOT be forwarded across media/redirect trust boundaries.

Logs/errors MUST use safe allowlisted operational fields. Passwords, codes,
cookies, tokens, authorization headers, signed URLs, raw payloads, email, filenames,
locations and raw record IDs MUST NOT appear even in debug output. Redact errors
before rendering, not after emitting them. The protected manifest MAY retain only
needed opaque identities, mapped paths/filenames, sizes, hashes, resource metadata and safe run
outcomes. It MUST exclude credentials, URLs, captions, GPS and EXIF copies.
Document the sensitivity, backup/encryption options and removal implications;
original media can itself contain private location metadata.

### IX. Practical File Handling and Simple Reruns

This is a best-effort personal downloader, not a transactional backup system.
Download to a uniquely named temporary file in the destination filesystem, check
HTTP/copy success and expected size when available, close it, then rename/publish
it to the final path. Never stream directly into an existing final media file.
Use ordinary platform file operations; do not require a custom journal, receipt
protocol, fsync choreography, file-identity recovery or power-loss guarantees.
If publication fails, report that item and retain the old final file where it
has not been replaced. A fully copied final file may survive a crash without a
matching manifest entry; reruns may rediscover it by filename and size.

Overwrite defaults to N for each invocation. Only explicit Y/--overwrite permits
replacement of eligible different-size local media; never overwrite internal
state, follow a symlink or escape the destination through a remote filename.
Keep deterministic filename/collision handling. Same filename and size can skip
as an unverified match, never as proof of identical content. No mandatory hashing
of existing or downloaded media is required; artifact checksums remain separate.

On handled failure/cancellation, try to delete only temporary files created by
this invocation. After a crash, stale partials may remain for manual removal;
do not reuse them or automatically delete unknown files. Reruns start interrupted
files at byte zero and compare final filenames/sizes. Byte-range resume, exactly-
once publication, rollback and automatic crash recovery are out of scope. Run
only one copy against a profile/destination at a time; robust multi-process
coordination and concurrent logout are not required.

### X. Best-Effort Progress and Honest Reporting

Try to enumerate the declared library scope and download the supported originals
that can be resolved. An isolated item/page failure must be reported, not hidden;
known usable items may still be downloaded after a later listing failure. Mark
such a run incomplete. Do not promise exhaustive enumeration, a remote snapshot,
all-format support or a byte-verified backup. Unsupported/missing originals and
filename/size skips must remain distinguishable from successful transfers.
Remote disappearance must never cause local deletion.

Use a small bounded retry policy for transient failures, honoring server delays.
Skip failed/unsupported items and continue where practical; stop for failed
account authentication, unusable destination, disk-full or other conditions that
prevent useful safe work. Do not bypass authentication or publish a failed copy.
Ctrl-C stops new work, attempts current temporary-file cleanup, and prints a safe
summary. Abrupt termination may lose recent progress/report entries.

The manifest is an advisory last-run/progress report, not a prerequisite for
copying or a durable transaction log. Save periodically and on normal exit on a
best-effort basis; warn if it cannot be saved. Offline status describes the last
saved report and may be stale. Do not claim current success from old metadata.
Dry-run, status, help and version remain write-free as specified.

Routine tests must use synthetic fixtures/local doubles; real-account validation
is manual and opt-in. Focus on ordinary transfers, prompts, filename/size choices,
privacy and common errors. Report untested cases/platforms honestly; exhaustive
crash matrices, fixed coverage thresholds and fully resolved protocol edge cases
are not personal-use release gates. Unknown protocol behavior still needs enough
evidence to implement the operation rather than invent it.

The no-argument menu asks username, no-echo password, destination and overwrite
[y/N]. Existing command mode remains. Credentials stay transient unless separately
opted in; local overwrite never grants account trust or password persistence.

## Security and Configuration

Credentials, tokens, private account data, logs, IDE settings, and machine-local
agent state MUST stay out of version control. Examples MUST use dummy values;
logs MUST redact sensitive values. Configuration MUST be validated and passed
explicitly to consumers. Preserve license choices and upstream notices.

Preserve generated ignores for local feature state and extension overrides.
Shared configuration MUST NOT be hidden by broad ignore rules. Publishing,
merging, and remote settings changes MUST remain within authorized scope. When
public GitHub security setup is authorized, enable supported secret scanning and
push protection, restrict Actions to reviewed dependencies, and disable workflow
PR approvals. Preserve stronger protections and report unsupported features or
missing access accurately.

Ignore IDE settings (including `.idea/`, `.vscode/` and applicable IDE files),
personal `.codex/` state, generated `*.log` files and local secrets; keep matching
IDE exclusions under `.github`. Preserve chosen exceptions and tracked files;
ignore maintenance MUST NOT silently untrack user content. If JSON ignores are
introduced later, required shared configuration and sanitized fixtures MUST have
narrow exceptions, with the actual allowed paths reported. The existing choice
to retain shared Spec Kit JSON takes precedence over a blanket `*.json` preference.
During future application setup, use the canonical `logs/` directory and retain
`logs/.gitkeep` if an empty tracked directory is needed; do not introduce an
alternate `log/` directory or commit runtime output.

## Development Workflow and Quality Gates

Work MUST proceed through specification, clarification as needed, planning,
tasks, analysis, implementation, and verification. Plans MUST check this
constitution before research and after design; justified deviations MUST appear
in complexity tracking. Preserve unrelated edits.

For code/tooling changes, run the README's applicable repository validation
commands before review. For an explicitly documentation-only task that forbids
test execution, perform static link/ID/consistency and diff/scope checks instead;
report that tests were not run. CI MUST run on
pull requests to and pushes to the actual default branch, using stable check
names, read-only tokens, a timeout, reviewed Actions pinned to full commit SHAs,
and checkout without persisted credentials. Exclude IDE folders from .github.

Configure remote required checks only when authorized, using the actual passing
check name and provider. Files and instructions alone MUST NOT be described as
enforced remote protections. Report missing tools, permissions, or services.

## Governance

### Guidance Responsibilities

Root AGENTS.md MUST focus on project structure, commands and local conventions,
referencing this constitution for durable rules. Feature requirements, design and
work items belong in specs/. Generated Spec Kit skills/templates/helpers MUST be
preserved; custom workflows belong in separate skills, without unnecessary agent
or subagent definitions. Preserve existing integration settings during preference
updates; source refresh, installation and integration migration are separate scope.

### Coding Guidance Provenance

This amendment adapts the local `.codex/AGENTS.md`,
`.codex/skills/initialize-go/SKILL.md`, its `EFFECTIVE_GO.md` reference,
`.codex/templates/go-project/AGENTS.md`, and the Spec Kit constitution principles.
The 2026-11-14 preference review also used the available
`.agents/coding-preferences` defaults, Go template and Spec Kit principles.
These rules are recorded here so shared governance does not depend on the ignored
personal `.codex/` directory. Project privacy, authentication, integrity and wire
contracts take precedence over generic coding preferences. In particular, do not
add remote observability, hide shared JSON, or force toolchain changes to match a
generic preference. A constitution amendment is not permission to implement the
downloader or refresh tools/reference sources.

### Amendment and Compliance Rules

This constitution governs project engineering decisions. Plans and reviews MUST
check compliance and document justified deviations. Amendments MUST record their
rationale, update the version and amendment date, and review affected templates,
skills, and active plans for compatibility. Preserve managed templates except
when an intentional tooling upgrade requires changes.

Use semantic versioning: MAJOR for incompatible governance changes or principle
removal, MINOR for new principles or expanded obligations, and PATCH for
clarifications with unchanged obligations. Preserve the original ratification
date. Operational commands and their validation limits belong in the README.

**Version**: 3.1.1 | **Ratified**: 2026-11-07 | **Last Amended**: 2026-11-14
