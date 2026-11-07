# Tasks: Personal iCloud Photos Archive

**Status**: initial implementation checkpoint. T001 setup is complete after local
Go 1.27.3 checks and four-target cross-compilation. Remaining tasks are pending or
partial; unchecked does not mean no code exists. The offline media/report core,
HTTP boundary primitive and command skeleton exist. Apple authentication/catalog,
interactive wizard, sessions and full orchestration do not work yet. See README.
No live account or GitHub workflow execution has been tested.
Inputs: [spec](spec.md), [research](research.md), [plan](plan.md),
[constitution](../../.specify/memory/constitution.md).

These tasks describe a future implementation authorized separately. Do not run
implementation commands merely because this document exists. IDs are stable;
dependencies refer to task IDs below. Every phase preserves read-only remote
access, explicit authentication, credential privacy, and honest completion.
All routine checks use synthetic fixtures/mocks and temporary directories.
Protocol review tasks do not grant permission to use a real account.

## Phase 1 — Repository and Protocol Validation

- [X] **T001** — Future setup: configure the required Go 1.27.3 toolchain and select the module path,
  package boundaries, build/format/vet/test commands, and local-only observability
  policy. Under a later implementation request, introduce go.mod with `go 1.27.3`
  and the module/first
  buildable package and immediately extend the existing local validation and CI
  with formatting checks, `go build ./...`, `go test ./...` and `go vet ./...`
  using Go 1.27.3 and required tags; prevent implicit toolchain upgrades. This baseline gate precedes all later coding;
  do not defer native checks to T022. Add no Go files during this documentation fix.
  Dependencies: none. Requirements: FR-001, FR-018, FR-021.
  Done when Go 1.27.3 is configured and four-target JSON/terminal/filesystem choices are reviewed and the same
  baseline Go checks pass locally and matching CI is configured from the first
  module change. Report GitHub execution separately; never claim an unrun CI
  check passed. This documentation update does not complete the setup task.
- [ ] **T002** — Protocol review: independently specify auth/session/challenge
  transitions and candidate auth/service/media authority admission policy, with
  pinned source attribution. Resolve source-review parts of OQ-001/OQ-006; leave
  live-only claims explicitly pending. Dependencies: T001.
  Requirements: FR-002, FR-003, FR-004, FR-018, FR-021.
  Done when synthetic protocol vectors and failure transitions are reviewable,
  no implicit trust/bypass exists, and every proposed network authority has evidence.
- [ ] **T003** — Protocol review: define pagination units, termination, joins,
  counts/change indicators, indexing and throttling from observed source. Author
  synthetic multi-page fixtures without captured account data. Dependencies: T002.
  Requirements: FR-006, FR-007, FR-014, FR-020, FR-021.
  Done when OQ-002/OQ-007 source findings and remaining manual checks are recorded,
  including duplicates, missing masters and failed middle pages. Snapshot/count
  proof is not required; enough evidence for the basic listing flow is sufficient.
- [ ] **T004** — Protocol review: document original resource role/owner selection,
  size/revision/checksum interpretation and fresh lookup/expiry behavior. Classify
  image/video types and provisional Live Photos/alternates explicitly.
  Dependencies: T003. Requirements: FR-008, FR-010, FR-021.
  Done when OQ-003/OQ-004/OQ-005 have evidence or explicit blockers; no all-format
  promise or invented checksum algorithm remains.

## Phase 2 — CLI, Privacy, and Session Boundaries

- [ ] **T005** — Implement future CLI parser, exit mapping, offline status boundary,
  cancellation context, and fakeable no-echo terminal prompts. Dependencies: T001.
  Requirements: FR-001, FR-002, FR-015, FR-016, FR-023.
  Done when unknown flags, absent terminal, invalid input and cancellation tests
  pass without secrets in arguments, environment or output (AC-001).
- [ ] **T006** — Implement separate injectable auth/service and media HTTP clients,
  reviewed endpoint/redirect admission, safe timeout/body limits and error/log
  allowlist. Disable external telemetry/exporters. Dependencies: T002, T005.
  Requirements: FR-017, FR-018, FR-021.
  Done when secret-canary and prohibited-egress tests pass (AC-010), including
  nested net errors, unexpected redirects and attempted auth-header forwarding.
- [ ] **T007** — Implement private optional session storage, a simple profile/account descriptor
  and idempotent local logout. Use platform state roots and owner-only credential access; reject
  known binding mismatches. Document one active invocation and no logout during a download. No
  profile/destination lock protocol or pre-network durable run marker is required.
  Dependencies: T005, T006. Requirements: FR-001, FR-004, FR-005, FR-012, FR-019.
  Done when basic opt-in storage/logout and private-access tests pass AC-014; local clearing is
  not server revocation or concurrent-process coordination.

- [ ] **T008** — Implement independent normal authentication/verification and optional
  command-mode session reuse. The wizard uses the supplied password for fresh authentication;
  unsupported challenges fail clearly. No automatic trust or password persistence.
  Dependencies: T002, T006, T007. Requirements: FR-002, FR-003, FR-004, FR-021.
  Done when ordinary success, invalid input, missing terminal and challenge/cancel cases pass
  AC-001/AC-014 without secrets in output.

- [ ] **T009** — Implement dry-run routing with ephemeral auth and zero writes or
  trust operations; finish offline status behavior. Dependencies: T005, T007, T008.
  Requirements: FR-013, FR-016.
  Done when a filesystem mutation spy and network recorder prove AC-013 before
  adding catalog/media orchestration.

- [ ] **T026** — Implement CWD-relative private runtime logging. Resolve
  `logs/icloud-dl.log` from launch CWD, create a missing directory, reject unsafe
  paths/permissions, fail 5 without fallback, and keep all readonly commands
  write-free. Dependencies: T005, T007. Requirements: FR-017, FR-019, FR-022.
  Done when AC-015 tests distinct CWDs, missing/unsafe/unwritable directories,
  existing safe logs, Unix modes/Windows ACLs and zero secret output.
- [ ] **T027** — Implement the local no-argument wizard: username, no-echo password, destination
  and overwrite [y/N], default N. Use its password for fresh authentication, show the actual
  originals path, preserve command-mode session reuse and separate persistence/trust consent.
  Dependencies: T005, T007, T008, T026. Requirements: FR-001, FR-002, FR-003, FR-012, FR-023,
  FR-024.
  Done when ordinary prompt order/validation/cancel/no-terminal cases pass AC-016 without
  credentials in logs. No profile lock or durable marker is needed.

## Phase 3 — Catalog and Resource Modeling

- [ ] **T010** — Implement discovery and the read-only source-readiness check. Stop if
  account/service access prevents useful work; do not invent endpoints or treat indexing failure
  as empty success.
  Dependencies: T003, T006, T008. Requirements: FR-001, FR-006, FR-012, FR-016, FR-020.
  Done when readiness/schema failure produces a useful safe summary/exit; no durable history
  marker is needed (AC-004, AC-011).

- [ ] **T011** — Implement straightforward paginated listing, duplicate detection and
  asset/master joins. Stop obvious loops and retain usable items when a later page fails.
  Downloading those known items is allowed; final summary says listing incomplete.
  Dependencies: T003, T010. Requirements: FR-006, FR-007, FR-020.
  Done when synthetic pages/shared masters/duplicates and one later-page failure pass
  AC-003/AC-004. Snapshot/count-proof reconciliation is out of scope.

- [ ] **T012** — Implement resource classification and role/lookup-owner model from
  reviewed findings. Keep unsupported/compound cases visible. Dependencies: T004,
  T011. Requirements: FR-007, FR-008, FR-012.
  Done when AC-012 distinguishes originals from thumbnails/edits and records every
  logical asset, including unresolved Live Photo companions as unsupported; other usable items
  can continue.
- [ ] **T013** — Implement just-in-time media URL lookup and bounded expiry refresh
  without URL persistence or worker prompts. Dependencies: T004, T006, T008, T012.
  Requirements: FR-008, FR-014, FR-017.
  Done when AC-005 passes and persistent denial returns an actionable category.

## Phase 4 — Simple Downloads and Reruns

- [ ] **T014** — Implement a small versioned advisory JSON progress/last-run report and optional
  filename-mapping history. Keep current outcomes in memory and try a snapshot every 25 finished
  items or 30 seconds, plus normal exit. No per-file journal, receipts, fsync choreography or
  required run marker.
  Dependencies: T007, T012. Requirements: FR-001, FR-010, FR-011, FR-012, FR-019.
  Done when missing/unreadable/unknown-schema reports do not block copying, unknown existing
  report data is not overwritten, save failure warns/produces exit 3, and status clearly shows a
  possibly stale last report (AC-011).

- [ ] **T015** — Implement stable safe filenames and deterministic suffixes for known remote
  collisions, including case/Unicode/Windows reserved names. Preserve optional saved mappings
  but allow reconstruction if history is missing. Show the actual originals subdirectory to the
  user.
  Dependencies: T012, T014. Requirements: FR-009, FR-019, FR-024.
  Done when representative traversal/symlink/duplicate-name tests stay inside the destination
  and do not silently replace another known asset (AC-003, AC-006). No adversarial
  concurrent-writer guarantee is required.

- [ ] **T028** — Implement the filename/size decision table: equal known size skips unverified,
  different size with N preserves/conflicts, Y permits a new staged replacement. Unknown size
  cannot justify skipping but an identified original may still download. Consent is in-memory
  for this invocation only; no persistent overwrite default or receipt.
  Dependencies: T012, T014, T015. Requirements: FR-009, FR-010, FR-012, FR-016, FR-024.
  Done when ordinary skip/N/Y/missing/unknown-size cases and dry-run behavior pass
  AC-006/AC-008/AC-017.

- [ ] **T016** — Implement ordinary temporary-file downloads on the destination filesystem,
  HTTP/copy/close checks, expected-size checks when available, and rename/publication after
  success. N never intentionally replaces an existing final; Y may replace eligible
  different-size media only after its new copy is ready. Report per-item publication/copy
  failure and continue where practical.
  Dependencies: T013, T014, T015, T028. Requirements: FR-008, FR-010, FR-011, FR-016, FR-019,
  FR-024.
  Done when a successful copy and representative HTTP/short-body/write/rename failures pass
  AC-006/AC-007/AC-017. No mandatory media SHA-256, transfer receipt, power-loss durability or
  platform transaction framework.

- [ ] **T017** — Implement best-effort cleanup of this invocation's temporary files on handled
  failure/cancel. Ignore leftover partials on reruns, start unfinished copies at zero and
  document manual stale-partial removal. Never delete unknown files automatically.
  Dependencies: T016. Requirements: FR-010, FR-011, FR-015, FR-016.
  Done when a normal interruption/cleanup and a rerun with stale partials pass AC-002; no
  persistent ownership/adoption/rollback protocol or crash-point matrix is required.

- [ ] **T018** — Implement simple reruns using final filename and size, with unverified-skip
  labels and re-download of missing finals. Do not depend on report completeness; a published
  final with no report entry can still match normally. Preserve local media absent from later
  listings.
  Dependencies: T004, T014, T017, T028. Requirements: FR-009, FR-011, FR-012, FR-016, FR-020,
  FR-024.
  Done when ordinary skip/conflict/Y/re-download cases pass AC-008/AC-017 and unknown same-size
  contents are never claimed hash verified.

## Phase 5 — Reporting and Useful Tests

- [ ] **T019** — Implement a small worker pool (default two, configurable 1-4), up to four
  transfer attempts total, short backoff/Retry-After handling, one expired-URL refresh and
  cancellation. Skip/report isolated failures; stop when authentication/storage prevents useful
  work.
  Dependencies: T011, T013, T017. Requirements: FR-014, FR-015, FR-016.
  Done when a transient failure, throttling, retry exhaustion and cancellation pass
  AC-005/AC-009. No elaborate scheduler/timing/recovery guarantees.

- [ ] **T020** — Implement useful progress/final counts and simple exits for partial listing,
  failed/unsupported items and report warnings. Successful earlier transfers remain useful after
  later failures. Status is only the last saved report, not current crash-proof history.
  Dependencies: T009, T014, T018, T019, T026, T027, T028. Requirements: FR-001, FR-012, FR-013,
  FR-015, FR-016, FR-020.
  Done when representative success/partial/cancel/error and failed-report-save paths pass
  AC-004/AC-008/AC-011/AC-013, without claiming full-library verification.

- [ ] **T021** — Run useful offline tests for ordinary auth/menu/list/copy/skip/overwrite,
  secret redaction, path safety, one interrupted file, reruns and common failures. Use synthetic
  responses/temp directories; no account requirement.
  Dependencies: T020. Requirements: FR-002, FR-003, FR-004, FR-005, FR-009, FR-010, FR-011,
  FR-017, FR-018, FR-019, FR-021.
  Done when important AC-001 through AC-017 flows have practical coverage and known gaps are
  reported. Do not require exhaustive crash/process/filesystem fault injection. Build/native
  smoke evidence is handled separately by T029.

- [ ] **T022** — Maintain T001's local/CI formatting/build/test/vet commands and document test
  results/known gaps. Coverage goals remain aspirational rather than a fixed release threshold.
  Dependencies: T021. Requirements: FR-021.
  Done when the useful offline tests and ordinary validation commands pass; no new remote
  settings or live-account requirement is introduced.

- [ ] **T023** — Document personal best-effort limits, originals subdirectory, filename/size
  skips, overwrite N/Y, CWD logs, stale partial removal, stale/missing reports, private session
  handling and native platform test status. Keep installation examples free of real account
  data.
  Dependencies: T020, T021. Requirements: FR-001, FR-003, FR-005, FR-011, FR-016, FR-019,
  FR-020, FR-022, FR-023, FR-024, FR-025.
  Done when help matches actual behavior and does not promise verified backup, reliable resume
  or crash recovery.

## Phase 6 — Optional Smoke Validation and Personal Readiness

- [ ] **T024** — MANUAL, OPTIONAL AND OPT-IN: the owner may try a small own-account library
  after the ordinary offline tests, confirming available auth/list/original-download behavior.
  Do not request credentials/codes in chat, export raw traffic, manufacture throttling or change
  account security.
  Dependencies: T002, T003, T004, T021, T022, T023. Requirements: FR-002, FR-003, FR-006,
  FR-007, FR-008, FR-014, FR-018, FR-020, FR-021.
  Done when observations/limitations actually checked are recorded without account data.
  Unverified edge cases may remain unsupported; this optional smoke check is not an exhaustive
  protocol-release gate.

- [ ] **T029** — Validate the existing tag-build workflow against real future Go source and
  inspect available target archives/checksums. Smoke-test native prompts/copies/private session
  state on platforms available to the owner; document untested builds instead of requiring four
  physical test systems.
  Dependencies: T001, T021, T022, T023. Requirements: FR-019, FR-021, FR-023, FR-024, FR-025.
  Done when build evidence actually obtained and AC-018/AC-019 smoke coverage/limitations are
  reported; never claim missing tests passed. Native ports may remain untested without important
  personal use. No tag push/Release/signing/remote-policy change is authorized by this task's
  existence.

- [ ] **T025** — Review practical personal-use readiness and limitations against the spec. Keep
  implementation tasks honest and list optional/unrun validation rather than requiring complete
  protocol/platform proof.
  Dependencies: T022, T023. Requirements: FR-001 through FR-025.
  Done when ordinary useful flows and retained privacy/read-only/path/overwrite safeguards are
  checked and remaining limitations are visible. T024/T029 may be pending with clear notes; do
  not auto-publish or commit.

## Requirement Coverage

| Requirements | Primary implementation/review tasks |
| --- | --- |
| FR-001 | T001, T005, T007, T010, T014, T020, T023 |
| FR-002, FR-003 | T002, T005, T008, T021, T024 |
| FR-004, FR-005 | T007, T008, T021, T023 |
| FR-006, FR-007 | T003, T010, T011, T012, T024 |
| FR-008 | T004, T012, T013, T016, T024 |
| FR-009, FR-010, FR-011 | T014, T015, T016, T017, T018, T021 |
| FR-012 | T007, T010, T012, T014, T018, T020 |
| FR-013 | T009, T020 |
| FR-014, FR-015 | T013, T017, T019, T020 |
| FR-016 | T005, T007, T010, T016, T017, T018, T020, T023 |
| FR-017, FR-018 | T002, T006, T013, T021 |
| FR-019 | T007, T014, T015, T016, T021, T023 |
| FR-020 | T003, T010, T011, T018, T020, T023, T024 |
| FR-022 | T026, T023 |
| FR-023 | T005, T027, T029 |
| FR-024 | T015, T028, T016, T018, T020, T029 |
| FR-025 | T029, T023 |
| FR-021 | T001, T002, T003, T004, T006, T008, T021, T022, T024 |

US1 is delivered through authentication, catalog and transfer phases; US2 through
state/recovery/reconciliation; US3 through consent, preview/status and logout.
The user flows share the retained basic safeguards; elaborate reliability is out of scope. Optional media formats/delta synchronization
are not hidden tasks in this release. No checklist item is completed by the mere
presence of these design documents.
