# Research: Private iCloud Photos Protocol

**Reviewed**: 2026-11-07 | **Status**: source review only; no Apple account used.
See [requirements](spec.md), [design](plan.md), and [validation tasks](tasks.md).

## Evidence and Attribution

The design uses observed third-party behavior because there is no public Apple
Photos-library API contract established for this use. These private web-service
observations are unsupported and can change without notice. Source inspection
shows what another client attempts; it does not prove current Apple behavior,
account compatibility, completeness, or permission to depend on any endpoint.
No Apple contract or guarantee is claimed.

Public source and revision history were available. Temporary shallow checkouts
were read at these exact revisions; no upstream code was copied into the project:

- icloud-photos-downloader/icloud_photos_downloader (icloudpd):
  `3caeae584c3d8bc6fcc51a1c9f7e5413b1422fd3`.
- rclone/rclone: `82cb84b2014bec6a167dc28eb53233b2d4abe1b3`.

The links below pin every protocol observation to those revisions. Full historical
analysis and live behavior were not verified. No captured upstream account
fixtures were imported. Future fixtures must be independently authored synthetic
examples, not copied account traffic. Function/field names below are research
pointers, not code to transplant; review licenses before any later reuse.

## Observed Facts in Source

### OBS-001 — Authentication and service discovery

icloudpd's `PyiCloudService.authenticate` first validates a reusable token, otherwise
calls `_authenticate_srp`, then `_authenticate_with_token`; authenticated metadata
contains `webservices`. `_get_webservice_url` and the `photos` property select the
Photos service. See
[base.py, authenticate and service lookup](https://github.com/icloud-photos-downloader/icloud_photos_downloader/blob/3caeae584c3d8bc6fcc51a1c9f7e5413b1422fd3/src/pyicloud_ipd/base.py#L294).
Rclone's `Client.authenticateSession` similarly reuses/validates state or calls
`Session.SignIn` and `AuthWithToken`; see
[api/client.go, authenticateSession](https://github.com/rclone/rclone/blob/82cb84b2014bec6a167dc28eb53233b2d4abe1b3/backend/iclouddrive/api/client.go#L126).
Its `Session.SignIn` implements an SRP exchange; `AuthWithToken` uses account
login metadata; see
[api/session.go, SignIn and AuthWithToken](https://github.com/rclone/rclone/blob/82cb84b2014bec6a167dc28eb53233b2d4abe1b3/backend/iclouddrive/api/session.go#L221).

### OBS-002 — Challenges, trust, and account-dependent access

Both clients expose two-factor verification and session trust. icloudpd's
`validate_2fa_code` calls `trust_session`; see
[base.py, validate_2fa_code and trust_session](https://github.com/icloud-photos-downloader/icloud_photos_downloader/blob/3caeae584c3d8bc6fcc51a1c9f7e5413b1422fd3/src/pyicloud_ipd/base.py#L830).
Rclone's `Validate2FACode`, `RequestSMSCode`, and `TrustSession` are in
[api/session.go, challenge handling](https://github.com/rclone/rclone/blob/82cb84b2014bec6a167dc28eb53233b2d4abe1b3/backend/iclouddrive/api/session.go#L647).
`ensurePCSCookies`/`acquirePCSCookiesFor` contain an additional device-approval flow;
see [api/session.go, PCS acquisition](https://github.com/rclone/rclone/blob/82cb84b2014bec6a167dc28eb53233b2d4abe1b3/backend/iclouddrive/api/session.go#L557).
This design deliberately differs: no implicit trust or account-setting changes.
If a required operation needs explicit consent that is declined, stop. Advanced
Data Protection and security-key compatibility are unverified, not bypass targets.

### OBS-003 — Photos service, scope, and indexing

Rclone's `NewPhotosService` reads `Webservices[WsPhotos]` and constructs a
Photos-specific database path; `buildSmartAlbums` distinguishes ordinary, hidden,
and deleted collections. See
[api/photos.go, NewPhotosService](https://github.com/rclone/rclone/blob/82cb84b2014bec6a167dc28eb53233b2d4abe1b3/backend/iclouddrive/api/photos.go#L514)
and [buildSmartAlbums](https://github.com/rclone/rclone/blob/82cb84b2014bec6a167dc28eb53233b2d4abe1b3/backend/iclouddrive/api/photos.go#L426).
icloudpd initializes `PrimarySync`, and its `PhotoLibrary.__init__` queries
`CheckIndexingState`, waiting for `FINISHED` with bounded checks; see
[services/photos.py, PhotoLibrary](https://github.com/icloud-photos-downloader/icloud_photos_downloader/blob/3caeae584c3d8bc6fcc51a1c9f7e5413b1422fd3/src/pyicloud_ipd/services/photos.py#L280)
and [PhotosService](https://github.com/icloud-photos-downloader/icloud_photos_downloader/blob/3caeae584c3d8bc6fcc51a1c9f7e5413b1422fd3/src/pyicloud_ipd/services/photos.py#L472).
Do not equate absent service or indexing-in-progress with an empty library.

### OBS-004 — Enumeration and asset/master relationships

icloudpd's `PhotoAlbum.photos` separates `CPLAsset` and `CPLMaster`, follows
`masterRef`, advances by assets, and retains the previous page's masters. Its
comment explains that multiple assets may share one master and that master-based
offsets can duplicate/lose assets. `_list_query_gen` uses `startRank` and
`resultsLimit`; see
[services/photos.py, photos and query generation](https://github.com/icloud-photos-downloader/icloud_photos_downloader/blob/3caeae584c3d8bc6fcc51a1c9f7e5413b1422fd3/src/pyicloud_ipd/services/photos.py#L596).
Its folder listing separately uses `continuationMarker`; see
[PhotoLibrary._query_folders](https://github.com/icloud-photos-downloader/icloud_photos_downloader/blob/3caeae584c3d8bc6fcc51a1c9f7e5413b1422fd3/src/pyicloud_ipd/services/photos.py#L363).
Thus album/folder markers must not be assumed to govern asset pagination.

Rclone's `buildPartitionQuery`/`fetchPhotosParallel` use rank partitions, and
`GetPhotos` includes cache/delta behavior not adopted for this milestone; see
[api/photos.go, rank queries and GetPhotos](https://github.com/rclone/rclone/blob/82cb84b2014bec6a167dc28eb53233b2d4abe1b3/backend/iclouddrive/api/photos.go#L1973).
Its `parsePhotoRecords` indexes an asset by master reference and `buildPhotos`
starts with a master-based ID; see
[parsePhotoRecords](https://github.com/rclone/rclone/blob/82cb84b2014bec6a167dc28eb53233b2d4abe1b3/backend/iclouddrive/api/photos.go#L1946).
These clients are not equivalent identity/completeness specifications. This
program preserves asset identity and treats unresolved joins as incomplete;
it must not copy an upstream skip-on-missing-record policy.

### OBS-005 — Resources and original media

icloudpd's `PhotoAsset.versions` examines resource fields in asset/master records
and reads size, `downloadURL`, and `fileChecksum`; see
[services/photos.py, versions](https://github.com/icloud-photos-downloader/icloud_photos_downloader/blob/3caeae584c3d8bc6fcc51a1c9f7e5413b1422fd3/src/pyicloud_ipd/services/photos.py#L1048).
Rclone's `buildPhotos` distinguishes `resOriginalRes`, Live Photo video
`resOriginalVidComplRes`, edited renditions, and `resOriginalAltRes`; see
[api/photos.go, buildPhotos](https://github.com/rclone/rclone/blob/82cb84b2014bec6a167dc28eb53233b2d4abe1b3/backend/iclouddrive/api/photos.go#L1815).
These are evidence for resource roles, not exhaustive format support. Exact
checksum algorithm, encoding, and representation coverage remain unverified.
A local SHA-256 establishes later local consistency, not authenticity of Apple
content or equivalence to an unverified remote fingerprint.

### OBS-006 — Fresh lookup and binary transfer

Rclone's `PhotosService.LookupDownloadURL` performs `records/lookup` for the
appropriate record/zone/resource field. `PhotosObject.Open` obtains that URL at
open time and issues a binary GET; see
[api/photos.go, LookupDownloadURL](https://github.com/rclone/rclone/blob/82cb84b2014bec6a167dc28eb53233b2d4abe1b3/backend/iclouddrive/api/photos.go#L2255)
and [icloudphotos.go, Open](https://github.com/rclone/rclone/blob/82cb84b2014bec6a167dc28eb53233b2d4abe1b3/backend/iclouddrive/icloudphotos.go#L492).
icloudpd's `download_media` uses partial files and range-oriented retries; see
[download.py, download_media](https://github.com/icloud-photos-downloader/icloud_photos_downloader/blob/3caeae584c3d8bc6fcc51a1c9f7e5413b1422fd3/src/icloudpd/download.py#L124).
Range support is not adopted here: restart interrupted resources from zero.
Exact URL lifetime and expiry status are not established by this review.

## Design Inferences, Not Service Guarantees

- INF-001: Discover endpoints after authentication instead of hardcoding a shard.
  Validate service/media hosts; a broad arbitrary URL from a response is unsafe.
- INF-002: Use serialized full catalog scans before transfers. Sequential scanning
  reduces complexity but does not create an atomic snapshot. Disclose observed changes when practical; snapshot/count proof is not required
  for best-effort personal copying.
- INF-003: Model logical asset ID separately from master/resource owner ID and
  resource role. Resolve associations explicitly across page boundaries.
- INF-004: Obtain fresh media URLs late and refresh boundedly on suspected expiry.
  Treat all URLs as secret; never persist or embed them in fixtures.
- INF-005: Preserve original bytes without promising every UTI/container. A
  compound/unsupported asset blocks unqualified completion rather than vanishing.

## Open Questions and Practical Validation

| ID | Unverified issue and required evidence | Tasks |
| --- | --- | --- |
| OQ-001 | Independent Go SRP/session implementation and terminal 2FA: synthetic vectors plus explicit own-account validation. Determine whether normal verification requires trust, and whether consent can be separated from persistence. Never assume a session works without trust. Security keys, regional accounts, and ADP may remain unsupported with actionable errors. | T002, T008, T024 |
| OQ-002 | Rank units, page termination, size limits, repeated assets, cross-page joins, and catalog mutations: derive synthetic fixtures from reviewed schema and validate small/multi-page controlled libraries. Determine usable counts/change indicators; no atomic snapshot guarantee. Unknown termination must be reported as incomplete listing; downloading usable known items is still allowed. | T003, T010, T011, T024 |
| OQ-003 | Original field selection, reliable size metadata, remote revision/fingerprint meaning, missing original resources, and checksum algorithm/encoding: validate if hash-verified reuse or remote-checksum enforcement is later implemented; neither feature is required for the best-effort first version. The explicitly requested filename/size skip policy may avoid a transfer, but is unverified content matching and must never be represented as checksum or revision validation. | T004, T012, T014, T018, T024 |
| OQ-004 | Live Photo still/video association and required companion completeness; RAW alternatives, bursts, edited assets, and exact supported formats. Validate separately before advertising these as supported. Initial release explicitly reports unresolved compound assets as unsupported/incomplete. | T004, T012, T024 |
| OQ-005 | Media URL expiry signals, resource lookup ownership, refresh behavior, and distinction from a new account challenge: verify bounded refresh without reusing stale secrets or prompting in workers. | T004, T013, T019, T024 |
| OQ-006 | Necessary Apple auth/service/media delivery host patterns and redirects, including CDN authorization. Document exact admitted authorities from source and opt-in validation; never permit arbitrary endpoints or broadly forward cookies. No production networking before admission policy is reviewed. | T002, T006, T024 |
| OQ-007 | Indexing state transitions, access restrictions, throttling/Retry-After, and library change detection. Source behavior differs; use bounded stop/defer behavior until validated. Missing count must not silently count as verified completeness. | T003, T010, T019, T024 |

## Fragility and Independent Implementation

Endpoint/schema changes, expired sessions/URLs, unavailable originals, incomplete
indexing, device challenges, regional/account policies, and throttling can break
this approach. Fail closed on unknown required fields and report recoverable work.
Do not reuse upstream authentication persistence, trust defaults, logging policy,
parallel scanning, delta sync, deletion behavior, or filename strategy as project
requirements. In particular, observation of a trust call does not authorize it.

No real credentials, tokens, signed URLs, personal identifiers, captured responses,
or media appear here. Source reviews and future observations must preserve this
boundary. Routine tests must not contact Apple. Account owners perform only
explicit manual checks; they never paste credentials/codes into agent chat.

## User-Directed Local Behavior and Distribution

The interactive wizard, CWD logs, filename/size skip policy, explicit local
overwrite and Debian/Windows/macOS build targets are product decisions, not claims
about Apple's protocol. Same-name/size files can contain different bytes; record
name-size-only assurance and do not infer verified originals. All original-source
size/type and URL questions above remain open. Cross-platform terminal input, private credential storage and ordinary file
operations benefit from available native smoke checks (T029), separate from
live-account protocol testing; untested ports are labeled rather than blocking
personal use on a working platform. No new endpoint or captured account
data is introduced by these requirements.

## Best-Effort Personal-Utility Decision

The owner explicitly prefers a simple useful downloader to backup-grade reliability.
Enough protocol evidence is needed to implement normal authentication, listing and
original download correctly, but unresolved optional formats/account conditions
may remain unsupported. Do not turn every open question into a release blocker.
Read-only access, ordinary Apple verification, private credentials, safe destinations
and honest limitations remain. Local journals/receipts, full-library count proofs,
mandatory media hashing and robust concurrent-process/crash recovery are not
requirements. Ordinary reruns compare final filename/size and restart partial work.
