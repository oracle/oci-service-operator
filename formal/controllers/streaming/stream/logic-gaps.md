---
schemaVersion: 1
surface: repo-authored-semantics
service: streaming
slug: stream
gaps:
  - category: bind-versus-create
    status: open
    stopCondition: "Formal semantics can branch between create, bind-by-id, and bind-by-name flows without routing through the legacy streams package."
  - category: list-lookup
    status: open
    stopCondition: "Formal semantics encode the current name plus optional pool or compartment filters and the lifecycle-sensitive matching used for create, update, and delete."
  - category: endpoint-materialization
    status: open
    stopCondition: "Formal semantics model the ACTIVE-only secret write that publishes the message endpoint and its delete-time cleanup."
  - category: status-projection
    status: open
    stopCondition: "Formal semantics either describe the handwritten OsokStatus projection or preserve it as an explicit legacy-only contract."
  - category: mutation-policy
    status: open
    stopCondition: "Formal semantics distinguish mutable fields from rejected changes, including the current partition and retention mismatch failures."
  - category: delete-confirmation
    status: open
    stopCondition: "Formal semantics capture or replace the current best-effort delete behavior that treats DELETING as sufficient for finalizer removal."
  - category: legacy-adapter
    status: open
    stopCondition: "stream_generated_client_adapter.go is removable because the formal runtime covers the current streams.StreamServiceManager behavior."
---

# Logic Gaps

## Current runtime path

- The generated `StreamServiceManager` is overridden by `stream_generated_client_adapter.go`, so all runtime behavior still delegates into `pkg/servicemanager/streams`.
- When `spec.id` is empty, the legacy manager lists streams by name plus optional pool or compartment filters to decide whether to create or bind.

## Repo-authored semantics

- Delete uses two different list-resolution paths: create or update ignores failed and deleted streams, while delete explicitly searches for failed, deleting, or deleted entries by name when no OCID is recorded.
- Update accepts tag changes and `streamPoolId`, but it rejects partition and retention mismatches as hard errors.
- Status projection is manual. The controller only records `OsokStatus`.
- When the stream reaches ACTIVE, the controller writes a secret that contains the `messagesEndpoint`; delete also attempts to remove that secret.
- Delete is only best-effort today. The legacy manager issues `DeleteStream`, checks once for `DELETING` or `DELETED`, and still returns success immediately.

## Why this stays on the legacy adapter

- The lifecycle-sensitive list matching and update rejection rules are not expressible in the current generic runtime contract.
- Secret materialization and best-effort delete confirmation need explicit formal semantics before the generated path can replace the legacy package.
