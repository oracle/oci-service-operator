---
schemaVersion: 1
surface: repo-authored-semantics
service: email
slug: emaildomain
gaps: []
---

# Logic Gaps

## Current runtime path

- `EmailDomain` uses the generated `EmailDomainServiceManager` and generated runtime client directly; there is no checked-in legacy adapter override.
- The checked-in runtime path stays service-local under `pkg/servicemanager/email/emaildomain` and `controllers/email/` while the shared generatedruntime provides the common CRUD orchestration.
- When no OCI ID is tracked, the generated path reuses a unique `ListEmailDomains` match on `compartmentId` plus `name` before create, and delete-time lookup can still resolve the resource when status has not recorded an OCID yet.

## Repo-authored semantics

- `EmailDomain` treats `description`, `freeformTags`, and `definedTags` as the in-place mutable update surface. Drift on `name` or `compartmentId` is replacement-only and the generated runtime rejects it explicitly instead of silently switching to a different OCI mutation API.
- Status projection is part of the repo-authored contract. The generated runtime merges the live OCI `EmailDomain` response into the published status fields, stamps `status.status.ocid`, and keeps OSOK lifecycle conditions aligned to the observed OCI lifecycle.
- The generated runtime follows create and update with read-based observation, requeues while OCI reports `CREATING`, `UPDATING`, or `DELETING`, and treats `ACTIVE` as the steady-state success target.
- Delete keeps the finalizer until `GetEmailDomain` or list fallback confirms the resource is gone or OCI reports terminal delete state. No Kubernetes secret reads or writes are part of this path.
