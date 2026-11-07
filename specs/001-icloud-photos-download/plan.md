# Implementation Plan: Best-Effort Personal iCloud Photos Downloader

**Status**: partial initial implementation; offline media/report core exists.
Apple authentication/catalog and CLI wizard remain pending. README records actual behavior.
**Targets**: Debian/Ubuntu Linux amd64, Windows amd64, macOS arm64 and amd64.
**Toolchain**: Go 1.27.3. Module: `github.com/jaysonhurd/icloud-dl`. Inputs:
[spec](spec.md), [research](research.md), [tasks](tasks.md), and
[constitution](../../.specify/memory/constitution.md), version 3.1.0.

## Scope and Design Approach

Build a convenient personal utility, not a transactional backup system. Keep the
menu, normal Apple authentication, direct original downloads, CWD logs, optional
session reuse and filename/size matching. Useful partial results are acceptable:
skip/report failed items, preserve good downloads, and let the owner rerun.

Do not implement transfer receipts, a recovery journal, per-file durable state,
mandatory media hashing, byte-range resume, rollback, snapshot consistency or
multi-process coordination. Keep temporary-file publication so failed transfers
do not become final files. Read-only iCloud access, secret handling, path safety,
explicit account verification and default-N overwrite remain mandatory. Follow
the constitution's Go conventions; tests focus on useful behavior rather than a
coverage percentage or exhaustive fault matrix. No runtime dependencies are added
by this document.

## Cohesive Go Responsibilities

These package boundaries now exist; protocol and orchestration work remains:

| Location | Responsibility |
| --- | --- |
| `cmd/icloud-dl` | Commands/menu, no-echo input, confirmation, cancellation and safe summaries. |
| `internal/archive` | Coordinate the personal download run; hold observed catalog, filename mappings and outcome counts in memory. |
| `internal/apple` | Independent Apple auth, endpoint discovery, pagination, asset/master association and fresh original URL lookup behind small interfaces. |
| `internal/state` | Private optional session state and a simple advisory JSON report; no locks, receipts or transaction log required. |
| `internal/media` | Safe filename/size decisions, temporary copies and normal platform rename/replacement. |

Inject a small HTTP doer and input/output boundaries for tests. Keep configuration
explicit and use contexts for requests/cancellation. Prefer standard Go libraries;
review terminal/crypto dependencies when needed rather than copying upstream code.
Only one invocation should use a profile/destination at a time. Supporting multiple
copies, logout during a download, or unrelated programs modifying the same files
is outside the first personal-use release.

## Main Flow

1. With no arguments, ask username, no-echo password, destination and overwrite
   [y/N], default N. Validate required inputs and confirm destination/overwrite
   without echoing secrets. The wizard uses the supplied password for fresh normal
   authentication. Command mode can reuse an opted-in session and prompts only
   when needed. Missing terminal for required input exits 4; cancellation exits
   130. Do not create state for help/version/status/dry-run.
2. For normal execution, create/open `CWD/logs/icloud-dl.log` privately. Failure to
   use the requested log location is a safe stderr error (5), not a silent fallback.
   Check the destination can be used, refuse symlink/traversal/internal-state
   targets, and reject a known account/library mismatch. Do not chmod an unrelated
   user folder to force a private-media policy. Credentials remain in private
   platform state, not an arbitrarily selected media drive.
3. Authenticate with normal Apple verification and consent; never silently change
   trust or account security. Discover the service and try the admitted read-only
   Photos-source readiness request. If authentication/readiness prevents useful
   work, explain and stop. No mandatory durable run marker precedes these calls.
4. Collect catalog pages until normal end or failure. Detect obvious loops and
   malformed responses, associate assets/masters, preserve distinct asset IDs and
   deduplicate repeated observations. On a later listing error, retain usable
   entries already collected and mark the listing incomplete. There is no proof
   of a complete/consistent remote snapshot and no requirement to refuse all
   downloads just because another page failed.
5. Prepare stable safe filenames and resolve collisions among observed assets.
   Use an existing readable report as helpful mapping history when available;
   downloading must not depend on the report. Compare final filename and expected
   size. Equal name/size skips unverified; different size with N is a conflict;
   Y permits a new staged copy and replacement. Missing finals are copied again.
6. Download supported resolved originals with a fresh URL and a small retry budget.
   Use two workers by default, configurable 1-4; a simple bounded worker pool is
   enough. Failed/unsupported items are counted and skipped while others continue.
   Stop for failed account access, disk-full, unusable destination or another
   fatal condition. Unknown required protocol/schema fields are not invented.
7. Periodically try to save the advisory report and save it on normal exit. Print
   observed/downloaded/replaced/skipped/failed/unsupported counts, listing/report
   warnings and a nonzero exit for known incomplete work. Do not call filename/
   size skips verified content or report old metadata as the current run result.

## Advisory Manifest and Status

Use a small versioned JSON file at `DESTINATION/.icloud-dl/manifest.json`, separate
from credentials. It is a useful report and optional mapping cache, not an
inventory guarantee or authority for automatic cleanup.

| Entity | Helpful fields |
| --- | --- |
| Run | Schema version, random run ID, times, account/library identity when known, declared scope, listing status, overwrite choice, counts and safe warning categories. |
| Asset/resource | Remote asset/master/resource identity needed for association, mapped path, expected size when known, observed actual size, outcome and safe failure category. |
| Outcome | downloaded, replaced, skipped-name-size-unverified, failed, conflict or unsupported. Optional hashes may be recorded only if actually computed; no hash feature is required. |

Keep passwords, codes, cookies, tokens, signed URLs, account email, GPS, captions,
EXIF copies and raw payloads out of this report. Mapped filenames/identifiers are
still private metadata. Protect the report/session/log files with Unix 0700/0600
or equivalent restricted Windows access. If private access cannot be provided for the report, warn/skip saving it; safe
media copying can continue. Use private new media files where the filesystem supports it; warn about privacy limits on a user-selected media drive
rather than claiming Unix permission bits work everywhere.

Hold current outcomes in memory. As a simple default, try a snapshot every 25
finished items or 30 seconds, and on exit. Write a temporary report and rename it
where practical; do not add fsync sequences, transaction receipts or per-item
journal writes. If a report write fails, warn and continue safe downloads; final
exit is 3 if the report could not be saved. A missing report is normal first-run behavior. A malformed/unreadable or
unknown-version report is ignored for download decisions with a warning; leave unreadable/unknown
existing report data untouched rather than silently replacing it. Known account
mismatch still refuses a merge. Offline `status` explains when no usable report
exists and that a saved report can be stale; it does not infer whether a newer
process ran or crashed. Normal logout clears local session/cache only and must be
used after other invocations finish. No cross-process invalidation is promised.

## Filenames, Temporary Copies and Reruns

Keep the existing layout `DESTINATION/originals/<sanitized-original-filename>`.
The selected directory is the archive root; media directly under that root is
not searched recursively or adopted. The menu/help must show the actual originals
subdirectory so the user knows where matching and downloading occur.

Remote names are untrusted leaf names. Reject traversal/separators/control or
reserved internal names; handle Windows reserved names and case/Unicode collisions
with deterministic identity suffixes. Missing/invalid names use a bounded identity-
based fallback with a known type extension. Never let two known remote assets
replace each other. Preserve saved mappings when available; if metadata is lost,
reconstructed mappings may differ, which is acceptable for this best-effort tool.

Before copying, inspect only an eligible mapped regular final file. Same name and
known expected size means skip, even with overwrite Y; do not claim a hash check.
Different size with N stays untouched and is counted as a conflict. With Y, copy
new bytes first. Unknown size cannot justify a skip, but a known original can be
copied and labeled size-unverified if no reliable length is available. An existing
final still needs explicit Y before replacement; N preserves it as a conflict. Unsupported
resource roles remain skipped/reported, not silently replaced with thumbnails.

Create a unique `.part` using ordinary safe temporary-file creation on the final
filesystem. Stream the successful response, check copy/close errors and expected
API/HTTP length when available, then rename/publish to final. Do not stream into
an existing final or delete it before the new copy is ready. Use practical native
create/replace operations while enforcing N versus explicit Y; if the operation
fails, report the item and continue where useful. Do not build a custom cross-
platform transaction system or promise durability across power loss/racing writers.

Try to remove only this invocation's temporary files after handled failure or
cancellation. A kill/power loss may leave partials; ignore them on reruns and let
the user remove them manually. They do not become final-file matches or block
unrelated work. No automatic reuse, broad cleanup, byte-range resume or receipt
recovery is required. If rename succeeded before a crash but the report was not
saved, the next run uses the existing final's name/size normally. Explicit overwrite
may have completed before a crash; no backup/rollback of old media is promised.

## HTTP, Retries, Cancellation and Errors

Keep auth/service and media clients separate. Use HTTPS/certificate checks,
reviewed Apple/authorized media destinations and bounded redirects. Never forward
account cookies to media hosts or send secrets to an arbitrary response URL.
Redact errors before display; do not log raw network errors carrying signed URLs.

Use ordinary request/header/connect timeouts and cancellation; large videos need
streaming without a short whole-file deadline. A simple 120-second no-progress
limit and bounded JSON response size are reasonable implementation defaults.
Initial transfer plus at most three retries is enough, with short 1/2/4-second
backoff. Honor Retry-After; defer/report a requested wait over 120 seconds rather
than retrying early. Refresh a suspected expired media URL once. Do not implement
unbounded nested retries, automatic challenge bypass or an elaborate scheduler.

Per-item HTTP/copy/size/publication failure is recorded and processing continues
where practical. Stop if useful safe work cannot continue (authentication loss,
disk-full, unusable destination or unsafe configuration). Ctrl-C stops scheduling,
cancels I/O, attempts current partial cleanup/report save and exits 130. Cleanup
or report writes are best effort; abrupt termination may lose recent report data.

Use FR-016 exits: 0 no known failures under the selected policy, 2 usage, 3 partial
results/report warning/conflicts, 4 authentication/challenge, 5 fatal local errors,
130 cancellation. Do not claim every item was discovered merely because exit 0
means there were no observed failures. Readonly preview/status commands never
create a log/report/session file.

## Testing and Practical Readiness

During future T001 setup, establish Go formatting/build/test/vet in the existing
local/CI tooling from the first module change. Use native Go testing, injected
HTTP doers, synthetic responses, fake terminal input and temporary directories.
Test normal auth/menu, secret redaction, a few pages/duplicates, filename choices,
N/Y decisions, a successful copy, common HTTP/size/write failures, cancellation,
and a normal rerun. A few targeted failure tests are enough; no exhaustive crash,
filesystem or concurrent-process matrix is required. The coverage goal is
aspirational, not a fixed release gate.

Protocol research must justify the operations actually implemented. Edge cases
may remain documented/unsupported without blocking personal use. The optional
own-account smoke check T024 remains manual and explicitly opt-in. Native smoke
tests on available systems are useful; label untested builds/known limitations
instead of requiring all four machines before the owner can try the utility.

## Tagged Build Workflow

Keep [tagged builds](../../.github/workflows/tag-build.yml): source checks, then
CGO-disabled builds of `./cmd/icloud-dl` for linux/amd64, windows/amd64,
darwin/arm64 and darwin/amd64 using exactly Go 1.27.3. Future go.mod must declare
`go 1.27.3`; builds must not silently select a newer toolchain. Unix archives are tar.gz, Windows zip;
archive SHA-256 sidecars and 30-day Actions artifacts remain. Debian means a Linux
binary archive, not .deb. No GitHub Release/signing/notarization is configured.
The workflow has local cross-compilation evidence for the initial development
executable; this does not prove live downloads or native platform behavior.
Build tooling may download Go/modules and upload requested binaries, but never
account credentials, photo metadata or media. This documentation update adds no
runtime implementation or dependencies.
