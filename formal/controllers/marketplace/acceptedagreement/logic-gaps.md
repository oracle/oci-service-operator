---
schemaVersion: 1
surface: repo-authored-semantics
service: marketplace
slug: acceptedagreement
gaps: []
---

# Logic Gaps

## Current runtime path

- `AcceptedAgreement` now uses the generated `AcceptedAgreementServiceManager`
  with a package-local generatedruntime wrapper under
  `pkg/servicemanager/marketplace/acceptedagreement`.
- Fresh reconciles reuse an existing accepted agreement before create by
  listing on exact `compartmentId`, `listingId`, and `packageVersion`, then
  matching `agreementId` plus optional `displayName` from the list response.
- Create and update are lifecycle-free in OCI. The package-local wrapper
  preserves generatedruntime request building, list reuse, mutation validation,
  and delete confirmation, while normalizing successful create and update
  responses straight to `Active` instead of waiting for a lifecycle token that
  the OCI read model never exposes.

## Repo-authored semantics

- Supported in-place updates are limited to `displayName`, `definedTags`, and
  `freeformTags`, matching `UpdateAcceptedAgreementDetails`.
- Drift for `agreementId`, `compartmentId`, `listingId`, `packageVersion`, and
  `signature` is replacement-only and is rejected against the currently bound
  agreement instead of allowing the controller to silently diverge from OCI.
- Because OCI `AcceptedAgreement` responses do not return the create-time
  `signature`, the controller mirrors the last applied value at
  `status.appliedSignature` so signature drift remains explicit and reviewable.
- Status projection remains required. Successful create, bind, update, and
  steady-state reads project the live OCI accepted agreement body into the CR
  status and stamp `status.status.ocid`.
- Delete is explicit and required. The controller retains the finalizer until
  `DeleteAcceptedAgreement` is followed by `GetAcceptedAgreement` returning OCI
  not found.

## Why this row is seeded

- The checked-in marketplace runtime now has an explicit bind-or-create answer,
  package-local create/update normalization for lifecycle-free responses,
  replacement-only drift checks for immutable inputs, and required delete
  confirmation without changing shared generatedruntime.
- No open formal gaps remain for the current generated controller and
  service-manager path.
