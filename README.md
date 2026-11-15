**WORK IN PROGRESS: THIS APPLICATION DOES NOT WORK YET. COMPATIBILITY WITH APPLE'S EVOLVING PRIVATE, UNSUPPORTED ICLOUD API REMAINS UNRESOLVED.**

# icloud-dl

An initial implementation and design kit for a best-effort personal Go CLI that downloads the owner's
own iCloud Photos originals directly to a local destination. Build targets are
Debian/Ubuntu Linux amd64, Windows amd64, macOS Apple Silicon arm64 and macOS Intel
amd64; native smoke checks and untested-platform limits will be documented. No work Mac,
cloud relay, or third-party media service is required. The Go module and offline copy/report core exist. The command now contains an **experimental, not live-validated Apple adapter**
for ephemeral SRP login, trusted-device verification, primary-library listing and
original download. No external Go dependencies
are used.

The service protocol is private, unsupported, and changeable. Research describes
observed icloudpd/rclone behavior, not an official Apple API contract. A future Go
implementation must be independent: no importing, vendoring, shelling out to,
or substantial copying from either upstream client.

The first milestone is normal authentication to one account, best-effort enumeration
of the primary non-hidden/non-deleted personal library, original still-image/video
retrieval, and an advisory local progress/last-run report. Reruns compare final filenames/sizes and restart unfinished files;
this is a one-time archive downloader, not continuous or two-way synchronization.
Read-only access forbids remote upload, deletion, editing, renaming, or moving.

Shared libraries/albums, hidden/recently deleted items, edited renditions, RAW
alternatives, bursts, web UI and additional architectures are outside the first milestone.
Live Photo companions remain a validation gate; unsupported resources must be
reported and prevent an unqualified complete-library claim.

## Initial Build Checkpoint

Use Go **1.27.3** with `GOTOOLCHAIN=local` and `GOEXPERIMENT=jsonv2`.
JSON v2 is explicitly enabled at the owner’s request; this SDK still gates it
behind an experiment. CI uses the same setting. The module is
`github.com/jaysonhurd/icloud-dl`. Build and check from the repository root:

```bash
export GOTOOLCHAIN=local
export GOEXPERIMENT=jsonv2
test "$(go env GOVERSION)" = go1.27.3
test -z "$(gofmt -l cmd internal)"
go build ./...
go test -cover ./...
go vet ./...
go build -o build/icloud-dl ./cmd/icloud-dl
./build/icloud-dl --help
```

Help/version and offline status work. On Linux, try a bounded manual run:

```bash
./icloud-dl download --destination /home/jayson/test --limit 5
```

The command prompts on the local controlling terminal for username and a no-echo
password, and for normal Apple verification when required. Do not paste passwords
or codes into agent chat. No session credentials are saved. Trusted-device prompt
delivery and any optional session trust operation each require explicit consent.
`--dry-run` does not create logs/reports/media or perform trust operations.
No real account has been tested by the agent; private endpoints may change and
some accounts require additional unsupported approval. Session persistence, SMS,
security-key/ADP repair flows and no-argument wizard remain unsupported. Optional
`--remember-session` is explicitly rejected. The current transfer runner is serial.

The internal copy engine has synthetic tests for staged publication, name/size skips,
explicit overwrite, symlink refusal, failed transfer cleanup, and cancellation.
The advisory report preserves unknown existing versions. Each mapped filename always
gets an identity suffix in this first pass, making mapping independent of listing order.
Reports contain private identifiers/paths: protect the destination (for example with
disk encryption); removing `.icloud-dl/manifest.json` removes only advisory history.
Leftover `.icloud-*.part` files after an abrupt kill require manual removal.

Pending: no-argument wizard, session persistence, retries/expired-URL refresh,
worker scheduling and 30-second report snapshots. Windows private logging/report
storage fails closed until current-user ACL support exists. Non-Linux credential
prompts fail safely until native no-echo input is implemented. All targets compile;
native macOS/Windows runtime behavior remains untested. Live Apple compatibility
is unverified. Initial session authentication and catalog code uses pinned source
observations (see research), not copied upstream implementations.

## Design Kit and Reading Order

1. [Constitution](.specify/memory/constitution.md): non-negotiable engineering,
   read-only, credential, privacy, integrity and failure-reporting rules.
2. [Specification](specs/001-icloud-photos-download/spec.md): user stories, stable
   requirement IDs, CLI behavior, scope and offline acceptance criteria.
3. [Research](specs/001-icloud-photos-download/research.md): pinned source findings,
   inferences, fragile assumptions and unresolved protocol validation gates.
4. [Plan](specs/001-icloud-photos-download/plan.md): cohesive Go responsibilities,
   session ownership, safe HTTP/filesystem boundaries, manifest and retry policy.
5. [Tasks](specs/001-icloud-photos-download/tasks.md): ordered future implementation
   and separate opt-in manual validation, traced to requirements. T001 setup is complete; remaining tasks are pending or partial.

This README is the entry document; these five linked documents complete the kit.
Repository working conventions are recorded in [AGENTS.md](AGENTS.md).
The existing constitution remains in its canonical Spec Kit location, avoiding a
second competing copy. The repository contains design documents and CI workflows. Implementing
the downloader requires a separate instruction.

## Security and Privacy Expectations

Passwords and verification codes are entered locally without echo, never in agent
chat, command arguments, configuration, or committed files. Only necessary session
credentials may be stored after explicit consent, in owner-only state directories
(0700) and files (0600) on Unix, or equivalent restricted current-user ACLs on
Windows. Trust/device decisions require explicit user consent;
convenience and retries never justify bypassing account controls.

Local logout clears stored session data, not Apple-side sessions or archive media.
A compromised session can expose account content; clearing local files is not
revocation or guaranteed secure erasure. Review account security through Apple if
compromise is suspected. Do not change security settings to make the program work.

No telemetry/analytics or remote tracing is allowed. Logs exclude account identity,
filenames, raw asset IDs, secrets and signed URLs. The manifest intentionally holds
sensitive opaque IDs, paths and sizes (plus hashes only if actually computed),
separate from credentials. Protect
archive storage/backups, consider disk encryption, and remove local metadata only
when the program is stopped. Removing it loses ownership/reuse history and does
not remove location/EXIF data embedded in original media.

Running `icloud-dl` without arguments will present a local terminal menu:
username, no-echo password, destination and overwrite **[y/N]**, default **No**.
This menu is specified, not implemented. Command mode remains available.

Normal runtime logs will use `logs/icloud-dl.log` beneath the working directory
where the program was launched. A missing log directory is created privately;
unsafe/unwritable locations fail clearly. Help/version/status/dry-run create no logs.

Existing media is compared by mapped filename and expected byte size. Equal name
and size is skipped and explicitly labeled **name-size-only; content unverified**.
Different-size files are preserved as conflicts by default; explicit Y/--overwrite
permits local replacement after the new temporary copy succeeds. Remote filename collisions receive
stable identity suffixes so separate assets cannot replace each other. This policy
never changes iCloud or bypasses path/credential/transfer validation. Publication
uses ordinary temporary copies and renames only after the copy succeeds.
A completed policy run containing unverified matches is not a byte-verified archive.
Pagination, unsupported resources and transfer failures are reported; usable items
can still be downloaded. This is not a guaranteed complete backup or transactional
sync tool.

For personal use, there is no transfer journal, receipt recovery, byte-range resume,
per-file durable checkpoint or exhaustive crash/platform release gate. A handled
interruption tries to clean up its temporary file; a crash may leave partials for
manual removal. The next run starts unfinished files over and can skip matching
finals even if the report was not saved. Report-save failures warn rather than
block safe downloading. Offline status is only the last saved report and may lag.
Use one copy at a time and finish/cancel downloads before logout. Useful tests and
optional small own-account/platform smoke checks are enough; list untested cases.

## Tagged Builds

[Tagged builds](.github/workflows/tag-build.yml) triggers on any tag pushed to
GitHub. It verifies the tagged source, then compiles/packages:

| Target | Go target | Package |
| --- | --- | --- |
| Debian/Ubuntu | linux/amd64 | tar.gz binary archive |
| Windows | windows/amd64 | zip containing icloud-dl.exe |
| Mac Apple Silicon | darwin/arm64 | tar.gz binary archive |
| Mac Intel | darwin/amd64 | tar.gz binary archive |

Each package has a SHA-256 sidecar in a separate Actions artifact retained for
30 days. Debian currently means a Linux binary archive, not an installable .deb.
The required toolchain and workflow use **Go 1.27.3**, CGO disabled, read-only
permissions and full-SHA-pinned official Actions. Source checks precede all builds;
no GitHub Release or signing/notarization is configured. References:
[Go setup action](https://github.com/actions/setup-go),
[artifact upload action](https://github.com/actions/upload-artifact), and
[tag push
events](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#push).

The Go module and entry point exist. Earlier local cross-compilation checks were
reported for all four targets; no GitHub workflow execution is established here.
The wizard, session persistence and several download acceptance criteria remain
incomplete. Current checks must be rerun using the pinned compiler.
Cross-compilation is separate from native OS safety tests. Existing remote allowed-
Actions policies may need separately authorized updates before the new pins can run.

## Spec Kit

The scaffold uses [GitHub Spec Kit](https://github.com/github/spec-kit), generated
with Specify CLI **1.3.2**, the Codex integration, and Bash helpers. Engineering
rules live in the [constitution](.specify/memory/constitution.md), version 3.1.1.

### Developer setup

Install Python 3.11+, [uv](https://docs.astral.sh/uv/), Git, Bash, and Codex, then:

```bash
uv tool install 'specify-cli==1.3.2'
specify version
specify check
```

Run the last two commands from the project root. Specify CLI 1.3.2 was confirmed locally on 2026-11-14. `specify check` checks tool availability,
not repository correctness.

Generated shared files belong in version control; a checkout containing them does
not need initialization. For initial setup without an existing `.specify/`, use:

```bash
specify init --here --integration codex --script sh
```

Review merge prompts in an existing directory. Follow the
[upgrade guide](https://github.github.io/spec-kit/upgrade.html) for intentional
upgrades. Open a new Codex session if generated skills are not visible.

### Project layout

```text
.specify/memory/constitution.md   Principles and governance
.specify/templates/              Upstream artifact templates
.specify/scripts/bash/           Generated workflow helpers
.specify/workflows/              Bundled workflow and registry
.agents/skills/speckit-*/         Expected Codex-managed skill locations
specs/001-icloud-photos-download/ Design spec, research, plan, and pending tasks
scripts/                        Local validation and regression tests
.github/workflows/validate.yml   Matching CI
```

### Current local integration finding

The 2026-11-14 `specify integration status` check reports eight missing managed
skills at their original locations, with zero modified managed files. Local
copies appear nested under `.agents/skills/speckit-tasks/`; those pre-existing
moves have been preserved. The Codex integration remains selected; no upgrade,
reinstallation or migration was performed. Skill visibility does not prove that
Spec Kit's managed-path checks pass. The available preference copy is under
`.agents/coding-preferences`; some linked supporting scripts are absent.

Keep root instructions concise and link to the constitution for durable rules.
Custom workflows belong in separate skills; do not edit generated skills or add
agent/subagent definitions merely to install preferences.

### Feature workflow

To amend the constitution, use Codex chat:

```text
$speckit-constitution Describe the proposed engineering principle changes.
```

For each feature, run these in chat one step at a time and review the results:

```text
$speckit-specify Describe the users, intended behavior, and acceptance criteria.
$speckit-clarify
$speckit-plan Describe the stack, architecture, and technical constraints.
$speckit-tasks
$speckit-analyze
$speckit-implement
$speckit-converge
```

Replace the descriptive text with actual requirements. Clarify before planning and
analyze before implementation. Repeat implementation and convergence until the
feature is verified. These are chat skills, not shell commands. The iCloud Photos
feature documents and partial application exist; review protocol evidence and
remaining acceptance tasks before extending implementation.

Commit shared `.specify/` infrastructure, `.agents/skills/`, and feature documents.
Generated JSON metadata is shared configuration and is eligible for version
control. Preserve `.specify/.gitignore`, which excludes local feature state,
extension overrides, and executable adapter packages. Personal `.codex/`, `.aws/`,
IDE settings, secrets, and logs are ignored.

## Repository validation

From the repository root, using Python 3.11+, Git, and Bash:

```bash
python3 -m unittest discover -s scripts -p 'test_*.py'
python3 scripts/check_repository.py
```

No downloaded Python packages are required. Checks cover skill names and required
metadata, inline local Markdown links and anchors, and Bash syntax. Tests exercise
valid inputs and meaningful failures. Metadata supports plain and quoted
single-line values; this is not a full YAML or Markdown parser. External URLs,
reference-style links, and fenced examples are not checked. In generated
`.specify/templates/` only, link destinations containing bracketed placeholders
are skipped; concrete links are still checked. Syntax checks do not execute or
verify the behavior of generated helpers and skills.

The `Repository validation` CI check runs the same commands on pull requests to
and pushes to `main`, with read-only permissions, a timeout, and SHA-pinned
checkout without persisted credentials. GitHub CI has not yet run. Remote required
checks and security settings require separate configuration and verification.

The Go 1.27.3 module now has formatting/build/test/vet checks in CI; run the
matching local commands in Initial Build Checkpoint above. GitHub execution has not
been performed locally.

## Documentation Update Verification

The iCloud Photos design update used static Markdown link/anchor and
requirement/task-reference checks plus a final scope/diff review. Tests were not
run for that update, and it included no implementation, live-account validation,
staging, or commits. Subsequent maintenance uses the repository validation commands
above unless explicitly restricted to static checks. Validator test results are
not evidence of downloader behavior or live protocol compatibility.

## Limiting a Test Download

The download parser accepts a file-selection cap:

```bash
./icloud-dl download --destination ./test-photos --limit 5
```

The command now prompts locally and attempts the private Apple flow. No live
account validation has been performed. The runner selects at most five distinct
supported originals; `--limit 0` (default) is unlimited. Skips and failures consume
selection slots, so fewer than five files may be written. Duplicates and unsupported
items do not consume slots. Preview uses the same cap with `--dry-run` and makes
no media requests. Enumeration may still scan the full catalog. A limited run is
not a complete-library download. Failures print safe categories rather than raw
responses, signed URLs or credentials. Live protocol incompatibilities remain
possible; do not interpret successful compilation as tested Apple compatibility.

## Preference and validation review (2026-11-14)

`agent update all` applies the available local preference copy; it does not fetch
preferences, refresh Effective Go, initialize/upgrade Spec Kit or change remote
settings. Private diagnostics and runtime-only CWD logging override generic
tracing/log-directory defaults. Shared JSON remains tracked without a blanket
JSON ignore. Existing JSON v2 experiment authorization and Go 1.27.3 stay fixed.
Repository checks inspect the current working tree, skip deleted tracked files
and directory entries, and resolve relative links beside symlinked source files.
They still report dangling links and do not repair pre-existing skill moves.
