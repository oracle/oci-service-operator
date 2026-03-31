---
schemaVersion: 1
surface: repo-authored-semantics
service: streaming
slug: stream
gaps: []
---

# Logic Gaps

## Current runtime path

- `Stream` now uses the generated `StreamServiceManager` and generated runtime client directly; there is no checked-in legacy adapter override.
- The generated runtime reuses an existing stream before create by listing on `name` plus the optional `streamPoolId` and `compartmentId` filters, then applies the formal lifecycle buckets to ignore delete-only matches during create or update while still resolving delete-time cleanup candidates.

## Repo-authored semantics

- Update accepts `streamPoolId`, `freeformTags`, and `definedTags`. The generated runtime rejects force-new drift for `name`, `partitions`, and `retentionInHours` against the live OCI resource body before any update call.
- Status projection is part of the repo-authored contract. The generated runtime merges the live OCI `Stream` response into the published status read-model fields while still stamping `status.status.ocid` and OSOK lifecycle conditions.
- Secret side effects are explicit repo-authored behavior. A narrow companion in the generated `stream` package writes the `messagesEndpoint` secret only after ACTIVE and deletes that secret during best-effort delete completion.
- Delete remains explicitly best-effort. The generated runtime issues `DeleteStream`, treats `DELETING` or `DELETED` as sufficient for finalizer removal, and uses delete-phase list lookup when no OCI identifier is already tracked.
