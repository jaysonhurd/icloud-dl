# Feature Specification: Personal iCloud Photos Archive

**Created**: 2026-11-07 | **Status**: Design draft; not implemented
**Toolchain**: Go 1.27.3, pinned for local development and CI.
**Targets**: Go CLI for Debian/Ubuntu Linux amd64, Windows amd64, and macOS
Apple Silicon (arm64) and Intel (amd64), one person's own Apple account.

Read the [constitution](../../.specify/memory/constitution.md),
[research](research.md), [plan](plan.md), and [tasks](tasks.md) together.
Requirement IDs below are stable; later amendments must not reuse retired IDs.

## Goal and Scope

**Reliability level:** best-effort personal utility. Prefer a small working
implementation over backup-grade durability, resumability or operational controls.


First milestone: authenticate to one account, make a best-effort listing of its
personal library, download supported originals to the selected local destination,
and save a helpful last-run/progress report. This is a personal utility; robust
backup guarantees, sophisticated resume and crash-proof state are not required. The protocol is
private, unsupported, and changeable; source observations are not an Apple API
contract. The downloader is not implemented; this kit also contains repository workflows.

For this release, personal library means non-hidden, non-deleted assets in the
primary private library. Hidden and recently deleted items are explicitly excluded
and the CLI must disclose this scope. No claim of backing up every item visible
in every Photos view is permitted. A separate specification is required to expand
scope. Upstream field names establish research leads, not format guarantees.

Non-goals: upload, delete, edit, rename or move remote content; local deletion to
mirror remote removals; two-way synchronization; continuous monitoring or delta
sync; album reconstruction; edited renditions; shared libraries/albums; RAW+JPEG
alternate resource preservation; burst expansion; web UI; Mac/cloud relay;
additional architectures beyond the four build targets. Originals are not transcoded. Live Photo
companions are provisional (OQ-004 in research); do not advertise full Live Photo
support or count a partially retrieved pair as complete.

## User Scenarios & Testing

### US1 — Archive originals (P1)

As the account owner, I can authenticate normally and archive my in-scope originals
to my computer without changing iCloud. An offline service double with multiple
pages and original resources can verify the complete journey.

Acceptance: AC-001, AC-003, AC-004, AC-005, AC-007, AC-010, AC-012, AC-016.

### US2 — Rerun interrupted work simply (P1)

As the owner, I can rerun after a failure and skip final files with matching names
and sizes, while restarting unfinished files from zero. Recent report entries and
stale temporary files may remain after a crash. Tests use ordinary existing files
and a mocked catalog; no transaction-recovery machinery is needed.

Acceptance: AC-002, AC-006, AC-008, AC-009, AC-011, AC-017.

### US3 — Inspect and control private state (P1)

As the owner, I can preview work, decline session persistence, and remove local
credentials while retaining my archive. Mock prompts and temporary directories
verify these controls without a real account.

Acceptance: AC-001, AC-010, AC-013, AC-014, AC-015, AC-016.

## Functional Requirements

| ID | Required behavior |
| --- | --- |
| FR-001 | Expose an interactive menu on invocation without arguments, plus login/download/status/logout commands and --help/--version. The menu asks username, no-echo password, local destination and Overwrite existing files? [y/N] in that order; typed input is terminal-only. Command mode accepts `download --destination PATH [--overwrite] [--dry-run]`; overwrite defaults to false on every invocation. Reject unknown options before network/filesystem changes. One account per profile/archive; refuse binding mismatches. See FR-023 for menu details. |
| FR-002 | Prompt for the account identifier and password on the local terminal as needed; password and verification-code input must not echo. Never accept passwords/codes in command arguments, environment configuration, agent chat, logs, or persisted state. Without a usable session and terminal, exit with actionable authentication-required output rather than hang. Minimize secret lifetime in memory without claiming guaranteed Go memory erasure. |
| FR-003 | Follow normal Apple authentication and two-factor verification. Trigger code delivery only after explicit user action; allow cancellation. Unsupported challenges stop with guidance. Never bypass challenges, change security settings, or automatically trust a device/session. Explain and obtain explicit consent before any trust operation required to continue; declining must stop safely if no supported alternative exists. |
| FR-004 | Session persistence is opt-in via `login --remember-session`; ordinary `download` uses an existing opted-in session or ephemeral authentication. Persist only necessary cookies/tokens and binding metadata, never the password/code or password-derived material. Use 0700 profile directories/0600 files on Unix or equivalent restricted current-user ACLs on Windows; reject symlinks, foreign ownership, and unsafe permissions. In-memory renewal may continue within the existing consent; new challenges return to the user. |
| FR-005 | `logout` clears selected local credentials/cache without deleting media or the report or mutating iCloud. It is idempotent and explains that local clearing is not server-side revocation, in-memory revocation or secure erasure. One active invocation per profile/destination is the supported usage; users must finish/cancel a download before logout or running another copy. Cross-process locking/invalidation guarantees are not required. Switching accounts uses a separate profile/destination; reject known binding mismatches. |
| FR-006 | Discover the private Photos service through authenticated metadata, validate HTTPS destinations/schema, and perform the read-only source-readiness check. Try the available pages until the normal end or an error; detect obvious loops and malformed pages rather than hang. A failed page is not an empty terminal page. Record incomplete enumeration, but downloading usable items already discovered is allowed. No remote-snapshot or provably exhaustive listing guarantee is required. |
| FR-007 | Preserve distinct asset IDs even when multiple assets reference one master. Deduplicate repeated occurrences of the same asset/resource only. Associate assets, masters, and resource roles across page boundaries; unresolved associations remain explicit failures. Enumeration completeness concerns logical assets, not the number of master records or files. |
| FR-008 | Select the available original resource for supported still images/videos and resolve a fresh URL just before transfer. Do not silently substitute a thumbnail, edited image, or transcoded rendition. Unknown resource schemas or missing originals are explicit unsupported/failed entries. Refresh an expired media URL within bounded retry policy. Signed URLs are transient secrets. |
| FR-009 | Use a stable remote asset/resource identity-to-path mapping with sanitized original filenames where available. Match only the mapped local filename and expected original byte size. Resolve remote filename collisions deterministically with identity suffixes, including case/Unicode/Windows reserved-name collisions; distinct assets never share a path. Same filename and size means skip as matched-existing with name-size-only assurance, never cryptographically verified. Different or unknown size is not a match; an identified original with unknown size may still be copied. Default N preserves a different-size existing file as a conflict; explicit Y/--overwrite permits post-copy replacement of eligible regular media files under the selected destination only. Reject symlinks, traversal, unsafe ownership and internal-state targets. See FR-024. |
| FR-010 | Copy each original into a uniquely created temporary file on the destination filesystem. Require a successful response and copy/close, checking expected byte length when available. Publish by ordinary platform rename/replacement after the copy finishes, preserving no-overwrite defaults and explicit Y. Do not stream into a final file or publish a known short/failed copy. No receipt journal, mandatory media SHA-256, fsync sequence, rollback, exactly-once or power-loss guarantee is required. If publication fails, report that item and preserve any existing final file not yet replaced. |
| FR-011 | Reruns use final filename/size matching; interrupted items restart at byte zero. Try to remove this invocation's temporary files on handled failure/cancellation. Abrupt termination may leave partials, which are ignored on later runs and may be removed manually. Never automatically reuse or delete an unknown partial. No byte-range resume, persistent transfer receipts, file-identity adoption or transactional recovery is required. A successfully published file missing from the report may be rediscovered by name and size. |
| FR-012 | Maintain a small versioned JSON progress/last-run report separate from credentials. Include account/library identity where known, resource identity/path, expected/actual size, downloaded/replaced/skipped/failed/unsupported outcomes, and safe counts/error categories. Save periodically and on normal exit on a best-effort basis, not per-file durable checkpoints. Failure to write the report warns and does not by itself stop otherwise safe downloads; it yields a nonzero result for an unsaved run report. A missing report is normal first-run behavior; an unreadable/unknown-schema report warns but does not block downloading. Use filename/size decisions and leave unrecognized existing report data untouched. Known account-binding mismatch still fails rather than merges accounts. Report entries may be lost after a crash. Skips are unverified; old status is only the last saved report, not proof of current progress. Dry-run saves nothing. |
| FR-013 | `download --dry-run` may authenticate ephemerally, enumerate and inspect local names/sizes, but never requests media bodies, writes logs/reports/session state or changes trust. Report planned work/uncertainty without claiming an archive was created. `status` is offline/read-only and labels the last saved report as potentially stale; no current-run history or live process state is promised. |
| FR-014 | Use a small bounded retry policy: up to four transfer attempts total, short backoff and one URL refresh for suspected expiry. Honor Retry-After; if the server delay is too long for this invocation, defer and report the item. Skip failed items and continue where practical. Do not loop on passwords/codes, unsupported challenges, permanent denial or unknown schemas. No sophisticated scheduler, automatic long-running recovery or retry-accuracy guarantee is required. |
| FR-015 | Ctrl-C stops new requests/work, cancels active operations, attempts cleanup of this invocation's temporary files and saves a report if practical, then prints a safe summary and exits 130. Immediate termination/power loss may leave partials and stale/missing reports. Do not report interrupted files as successfully transferred. |
| FR-016 | Use exits 0 for a run with no known failures under filename/size comparison, or successful control/preview; 2 usage; 3 partial listing, per-item failures/unsupported items/conflicts or failure to save the report; 4 authentication/challenge; 5 fatal local configuration/storage/log errors; 130 cancellation. Isolated transfer/size/publication failures may continue with other items; unusable destination/disk-full stops useful work. For simultaneous errors use 130, then 5, then 4, then 3. Summaries give downloaded/replaced/name-size-skipped/failed/unsupported counts and listing/report warnings without implying a verified full-library backup. |
| FR-017 | Logs/errors must exclude passwords, codes, cookies, tokens, authorization headers, signed URLs, account email, filenames, locations, raw record IDs, and unfiltered payloads. Allow only aggregate counts, durations, operation/error categories, HTTP status, and random local run IDs. Redact at error boundaries and use an allowlist, including in debug mode. |
| FR-018 | No telemetry, analytics, remote tracing/exporters, crash reporting, or transmission of media/metadata to third parties. Contact only Apple service endpoints and validated Apple-authorized media delivery endpoints needed for the requested operation. Never forward auth credentials to media hosts or unapproved redirects. Endpoint admission evidence must be validated before live use. |
| FR-019 | Keep credentials and managed private state/logs owner-only: 0700 directories/0600 files on Unix or equivalent restricted Windows access. Use private permissions for new files where supported; never silently chmod unrelated existing media folders. Existing user-selected media folders can be used with a warning if storage cannot enforce private media permissions; do not put reusable credentials there. Never follow symlinks or escape the destination through remote paths. Report paths/names/sizes are private metadata; exclude GPS, album names, captions, account email, EXIF copies, URLs, tokens and raw responses. Native file-copy/rename failures are reported; no custom filesystem durability protocol is required. |
| FR-020 | Each download makes a fresh best-effort enumeration. This is a one-way personal copy utility, not continuous or two-way sync. Preserve local media when a remote item disappears. Keep successfully copied items when later enumeration/downloads fail; report the incomplete result so the user may rerun. No snapshot consistency, count reconciliation proof or guaranteed discovery of every remote item is required. |
| FR-021 | Independently implement Apple protocol handling in Go. Do not import, vendor, execute, or copy substantial code from icloudpd/rclone. Record precise research sources and uncertainty; routine tests use synthetic fixtures and local doubles, with no account or Apple network dependency. Real-account validation is manual and opt-in. |
| FR-022 | Resolve runtime logs as `CWD/logs/icloud-dl.log` from the launch working directory. For normal login/download/menu/logout execution, create the missing logs directory with private permissions before opening the log; reuse only a safe directory/file. Never fall back to a different location silently; failure exits 5 with a safe stderr error. Do not create logs in the executable/repository directory merely because the binary lives there. Help, version, offline status and dry-run create no logs or directories. |
| FR-023 | No-argument invocation validates terminal availability, then asks username, no-echo password, destination and overwrite [y/N]. Blank/N/n means No; Y/y means Yes; invalid answers re-prompt. Inputs stay in memory; confirm destination/overwrite without logging inputs. The wizard uses its supplied password for fresh normal authentication; command mode may reuse an opted-in session. Cancel exits 130 without starting network work; missing terminal exits 4. Normal Apple verification and separate session/trust consent remain. No profile-lock or pre-network durable-run-record requirement applies. |
| FR-024 | Use expected original size when available. Same mapped name and size skips unverified, even with Y. Different size requires current-run Y/--overwrite for replacement; N leaves a conflict. Unknown size cannot justify a skip; an identified original may still be copied with HTTP/copy checks and size-unknown reporting. If its mapped final already exists, N preserves it and Y is required to replace it. Remote name collisions receive identity suffixes rather than replacing each other. Keep overwrite consent in memory for this invocation, not a persistent global default or receipt. Dry-run previews only. |
| FR-025 | A GitHub tag-push workflow verifies source then compiles CGO-disabled binaries for linux/amd64 (Debian/Ubuntu), windows/amd64, darwin/arm64 and darwin/amd64, packages them and SHA-256 checksums, and uploads per-target Actions artifacts. Any pushed tag triggers it. Use exactly Go 1.27.3, consistent with the future go.mod declaration, SHA-pinned Actions, read-only permissions, timeouts and checkout without persisted credentials. Missing Go module/entry point or a failed verification/build fails clearly; no placeholder executable or silently successful skip. Compilation is not native runtime validation. |

## CLI Contract

Proposed first-release forms (design only; no commands are implemented):

```text
icloud-dl                         # interactive wizard
icloud-dl login [--profile NAME] [--remember-session]
icloud-dl download --destination PATH [--profile NAME] [--workers 1..4] [--overwrite] [--dry-run]
icloud-dl status --destination PATH [--profile NAME]
icloud-dl logout [--profile NAME]
icloud-dl --help
icloud-dl --version
```

The default profile is `default`; names contain 1-64 ASCII letters, digits,
underscores or hyphens only. Credentials live below
`$XDG_STATE_HOME/icloud-dl/profiles/NAME`, or
`$HOME/.local/state/icloud-dl/profiles/NAME` when unset, on Linux. macOS uses
`~/Library/Application Support/icloud-dl/profiles/NAME`; Windows uses
`%LOCALAPPDATA%\icloud-dl\profiles\NAME`.
Require an absolute XDG state path. These environment variables select storage,
never supply passwords/codes. Account identity is prompted locally, not embedded
in profile paths. A profile cannot silently change its bound account.

Without remember-session, login only verifies ephemeral access and explains that
no reusable session credentials were saved; noncredential profile identity metadata may remain. Remember-session permits credential persistence but
is not blanket consent for device trust: explain and request that separately.
A new session after logout may reuse the same profile/account binding; a different
account needs a different profile and archive. Status reads the destination's
report without opening credential files; a missing report is reported plainly.
Logout does not require a destination. There is no remote delete, trust-bypass or
background-sync option. Local
overwrite is an explicit per-run choice, default N; it never mutates iCloud. Help and version
work offline without
creating state. Workers default to two; invalid values are usage errors.

Run one copy per profile/destination at a time, and finish/cancel downloads before
logout. Concurrent processes are unsupported; no complex locking or invalidation
protocol is needed. Status shows only the advisory last saved report. The wizard
uses fresh credentials; explicit command mode can reuse previously opted-in
session state. Missing progress metadata never prevents an ordinary filename/size
rerun. Password, verification, read-only access and overwrite safeguards still apply.

## Acceptance Criteria

All routine criteria use synthetic identities, mocked HTTP responses, fake prompts,
an injected clock/random source, and temporary files. No credentials or captured
account data enter this repository. Manual protocol validation is separate.

| ID | Scenario and measurable expected outcome | Requirements |
| --- | --- | --- |
| AC-001 | Valid/invalid/absent password and 2FA inputs: no echo or persistence; missing terminal exits 4; a new challenge pauses for explicit action and cancellation exits 130. No automatic trust call. | FR-002, FR-003, FR-004, FR-015 |
| AC-002 | Interrupt a copy: it is not published as final; handled interruption attempts to remove its temporary file. Simulate a leftover partial and a published final missing from the report: rerun ignores the partial, restarts missing items at zero and may skip the final by name/size. No journal/receipt/crash-recovery protocol is tested or required. | FR-009, FR-010, FR-011, FR-012, FR-016 |
| AC-003 | Three pages contain a repeated asset, two distinct assets sharing a master, and an association crossing pages: all distinct assets survive exactly once; required resource totals match logical roles. | FR-006, FR-007, FR-012 |
| AC-004 | A middle page fails, repeats without progress or has an unsupported schema: report incomplete listing and a nonzero result. Usable assets found earlier may still download; already copied files stay intact. Failed listing is never treated as successful empty-library enumeration. | FR-006, FR-015, FR-016, FR-020 |
| AC-005 | A media URL expires: try one fresh lookup inside the bounded policy, then skip/report if still unavailable. Do not persist or log URLs, and do not hang on repeated failure. | FR-008, FR-014, FR-017 |
| AC-006 | Test malicious names, symlinks and duplicate remote filenames: no destination escape or silent replacement of another asset. N preserves existing different-size media; Y permits replacement only after a successful temporary-file copy. Ordinary publication failure is reported; adversarial concurrent writers and exhaustive race/crash points are outside the supported single-invocation usage. | FR-009, FR-010, FR-019 |
| AC-007 | HTTP error, interrupted copy, close error or known expected-size mismatch does not publish the failed copy. After the small retry budget, report/skip the item and continue when practical, exiting 3 if any item failed. Disk-full/unusable destination stops the run with 5; cancellation uses 130. No mandatory hash/checksum algorithm or full fault-injection matrix is required. | FR-008, FR-010, FR-016 |
| AC-008 | Equal name/size skips unverified. Different size with N preserves/conflicts; Y stages a new copy before replacing. Missing media is copied again. No report is a normal first run. An unreadable report warns but does not prevent filename/size reruns; do not overwrite unknown report data. Same-size wrong content is not detected by this policy and is not claimed verified. | FR-009, FR-011, FR-012, FR-016 |
| AC-009 | Mock a transient request failure and a throttle response: retry within the small limit and honor Retry-After or defer. Invalid codes/permanent errors do not loop. Cancellation stops waiting promptly. Exhaustive scheduler timing and multi-process tests are not required. | FR-014, FR-015 |
| AC-010 | Secret canaries in credential/header/URL fields and nested errors never appear in captured stdout/stderr/logs/manifest. Mapped filenames may appear only in protected manifest paths, never diagnostic logs; they are not a permitted location for credential fields. Any attempt to contact an unapproved host or remote exporter fails before sending data. | FR-017, FR-018, FR-019 |
| AC-011 | A failed/partial run reports its actual counts/warnings even if an earlier saved report succeeded. Periodic/exit report writes are attempted; failure warns and does not block safe remaining transfers. Offline status explicitly labels its last saved report as potentially stale. A crash can lose recent report entries. There is no mandatory pre-network marker or transactional progress history. | FR-001, FR-012, FR-016, FR-020 |
| AC-012 | Supported image/video originals retain exact served bytes. Missing originals, unrecognized types, and an unresolved Live Photo companion remain visibly unsupported/incomplete, never silently downgraded. | FR-008, FR-010, FR-016 |
| AC-013 | Dry-run produces a useful plan with zero filesystem mutations, zero media requests, zero trust operations, and no persisted session refresh. Status makes zero network requests. | FR-013 |
| AC-014 | Opt-in session persistence stores no password/code and uses private access. Logout twice clears available local credentials without touching media/report and explains revocation limits. Tests assume one active invocation; concurrent logout/refresh races and cross-process guarantees are not required. | FR-004, FR-005, FR-019, FR-016 |
| AC-015 | Run from two distinct temporary working directories with logs absent: normal execution creates logs/icloud-dl.log under each CWD with private access, never beside the executable. Unsafe/unwritable log locations fail 5 with no secret output. Help/version/status/dry-run create nothing. | FR-017, FR-019, FR-022 |
| AC-016 | Feed the wizard fake terminal input: it prompts username/password/destination/overwrite in order, masks password, treats blank/N as No, accepts Y and re-prompts invalid answers. Cancellation or missing terminal sends no network traffic; no input/secret is logged. Normal authentication/2FA and separate persistence consent still apply. | FR-001, FR-002, FR-003, FR-012, FR-023 |
| AC-017 | Check equal-name/size skips, different-size N preservation, Y staged replacement, name collisions and unknown-size copies. A failed new copy never replaces an existing final. Dry-run writes nothing. Rerun behavior follows final name/size matching without receipts or automatic rollback. | FR-009, FR-010, FR-011, FR-012, FR-016, FR-024 |
| AC-018 | A pushed tag selects all four build targets; source verification failure blocks artifact production. Missing source fails with an actionable message. Each successful target artifact contains the expected executable, archive and valid SHA-256 checksum; packaging failures fail the run. Check local workflow structure/scripts now; real compilation/CI execution requires future application source. | FR-021, FR-025 |
| AC-019 | Smoke-test prompts, filename rules, private credential/log state and ordinary copies on available native platforms. Label untested ports/limitations in the README and release notes. Passing every native platform/crash scenario is not a personal-use release gate; do not claim unperformed tests passed. | FR-019, FR-023, FR-024, FR-025 |

## Key Entities and Success Criteria

An account profile owns an authenticated session; assets reference originals and
mapped local paths. The optional/advisory report summarizes observed outcomes;
it is not an authoritative record of every published file or a crash-recovery log.

Personal-use readiness means the ordinary auth/menu/list/copy/skip/overwrite flows
and important privacy/path/error cases have useful offline tests, plus an opt-in
small own-account check when the owner chooses. AC-001 through AC-019 describe
expected behavior, not a demand for an exhaustive reliability/platform matrix.
Keep unresolved protocol questions and unsupported formats visible; do not invent
required request behavior, but every edge case need not be settled before useful
personal use. Do not claim unrun tests or a verified complete backup. Binary builds
still target all four platforms; report native tests actually performed.
