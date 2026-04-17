---
schemaVersion: 1
surface: repo-authored-semantics
service: email
slug: sender
gaps: []
---

# Logic Gaps

## Current runtime path

- `Sender` now uses the generated `SenderServiceManager` and generated runtime
  client directly; there is no checked-in legacy adapter override for this
  resource.
- The generated runtime reuses an existing sender before create by listing on
  exact `compartmentId` plus `emailAddress`, then filtering out delete-only
  matches through the formal lifecycle buckets.
- Delete confirmation resolves tracked OCI identities through `GetSender`; when
  no `status.status.ocid` is recorded, the runtime falls back to `ListSenders`
  with the same identity filters before issuing `DeleteSender`.

## Repo-authored semantics

- Ready state maps to OCI `ACTIVE`, while `CREATING` remains provisioning and
  `DELETING` keeps the controller in terminating finalizer-retention state.
- Supported in-place updates are limited to `definedTags` and `freeformTags`,
  matching the `UpdateSenderDetails` SDK surface.
- Drift for `compartmentId` and `emailAddress` is replacement-only and is
  rejected against the current tracked resource instead of creating a second
  sender behind the controller's back.
- Create and update use provider-helper-backed read-after-write follow-up
  through `GetSender`, so the controller settles on the live OCI `Sender` body
  before reporting steady state.
- Status projection remains required. The generated runtime merges the live OCI
  `Sender` response into the published status read-model fields and stamps
  `status.status.ocid`.
- Delete is explicit and required. The controller retains the finalizer until
  `DeleteSender` succeeds and follow-up observation confirms `DELETED` or OCI
  no longer returns the sender.

## Why this row is seeded

- The checked-in generated controller and service-manager path now has an
  explicit bind-or-create answer, mutable-vs-replacement update policy, and
  required delete confirmation for `email/Sender`.
- No open formal gaps remain for the current generatedruntime contract.
