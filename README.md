# icloud-dl

An initial implementation and design kit for a best-effort personal Go CLI that downloads the owner's
own iCloud Photos originals directly to a local destination. Build targets are
Debian/Ubuntu Linux amd64, Windows amd64, macOS Apple Silicon arm64 and macOS Intel
amd64; native smoke checks and untested-platform limits will be documented. No work Mac,
cloud relay, or third-party media service is required. The Go module and offline copy/report core exist. **This checkpoint cannot yet
authenticate to Apple or download an iCloud library.** No external Go dependencies
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

Use Go **1.27.3** with `GOTOOLCHAIN=local`. The module is
`github.com/jaysonhurd/icloud-dl`. Build and check from the repository root:

```bash
export GOTOOLCHAIN=local
test "$(go env GOVERSION)" = go1.27.3
test -z "$(gofmt -l cmd internal)"
go build ./...
go test -cover ./...
go vet ./...
go build -o build/icloud-dl ./cmd/icloud-dl
./build/icloud-dl --help
```

Help/version and offline status work; login/download return exit 4 with an explicit
not-implemented error. No-argument invocation does not request credentials while
the authentication adapter is missing. The design below describes intended behavior,
not completed functionality. `--dry-run` currently fails at that same protocol boundary
without creating files. Unsupported options are never evidence of working behavior.

The internal copy engine has synthetic tests for staged publication, name/size skips,
explicit overwrite, symlink refusal, failed transfer cleanup, and cancellation.
The advisory report preserves unknown existing versions. Each mapped filename always
gets an identity suffix in this first pass, making mapping independent of listing order.
Reports contain private identifiers/paths: protect the destination (for example with
disk encryption); removing `.icloud-dl/manifest.json` removes only advisory history.
Leftover `.icloud-*.part` files after an abrupt kill require manual removal.

Pending: independent SRP and 2FA, service/host admission evidence, paginated catalog,
fresh URL lookup, no-echo wizard, session persistence/logout, bounded retries/workers,
30-second reporting, and end-to-end orchestration. Windows private logging/report
storage fails closed until current-user ACL support exists. Linux/macOS permission
checks exist; native macOS and Windows runtime behavior has not been tested.
`internal/apple.Guard` is a boundary primitive, not a production-ready Apple client;
its transport must disable automatic redirects using the supplied redirect policy.
No live account was used. Build artifacts are development checkpoints.

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

There is no go.mod or cmd/icloud-dl yet, so the workflow currently fails clearly
at its application-source prerequisite instead of uploading placeholders. No build
has been run on GitHub. Runtime menu/logging/download changes remain future tasks.
Cross-compilation is separate from native OS safety tests. Existing remote allowed-
Actions policies may need separately authorized updates before the new pins can run.

## Spec Kit

The scaffold uses [GitHub Spec Kit](https://github.com/github/spec-kit), generated
with Specify CLI **1.3.2**, the Codex integration, and Bash helpers. Engineering
rules live in the [constitution](.specify/memory/constitution.md), version 3.1.0.

### Developer setup

Install Python 3.11+, [uv](https://docs.astral.sh/uv/), Git, Bash, and Codex, then:

```bash
uv tool install 'specify-cli==1.3.2'
specify version
specify check
```

Run the last two commands from the project root. Specify CLI 1.3.2 is installed
persistently through uv on this machine. `specify check` checks tool availability,
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
.agents/skills/speckit-*/         Nine generated Codex skills
specs/001-icloud-photos-download/ Design spec, research, plan, and pending tasks
scripts/                        Local validation and regression tests
.github/workflows/validate.yml   Matching CI
```

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
feature documents now exist; review their research gates before any implementation.
No application has been generated.

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
