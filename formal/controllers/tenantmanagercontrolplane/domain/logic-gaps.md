---
schemaVersion: 1
surface: repo-authored-semantics
service: tenantmanagercontrolplane
slug: domain
gaps: []
---

# Logic Gaps

## Current runtime path

- `Domain` keeps the generated controller, service-manager shell, and
  registration wiring, but the reviewed runtime contract is finalized in
  `pkg/servicemanager/tenantmanagercontrolplane/domain/domain_runtime_client.go`.
- Create is work-request-backed. `CreateDomain` returns a concrete `Domain`
  body plus `opc-work-request-id`, reconcile stores the create work request in
  `status.status.async.current`, and success resumes through `GetWorkRequest`
  followed by a live `GetDomain` reread.
- Update is direct-body and synchronous in this rollout. `UpdateDomain`
  returns a concrete `Domain` body, so the runtime projects that response
  directly into status and does not require a follow-up work-request or
  read-after-write pass.
- Bind resolution is exact and paginated. When no OCI identifier is tracked,
  the runtime paginates `ListDomains`, maps the request `name` filter from
  `spec.domainName`, and adopts only a unique exact match on
  `compartmentId + domainName`, optionally narrowed by `spec.lifecycleState`
  and `spec.status`. Duplicate matches fail instead of binding arbitrarily.
- Mutable drift is limited to `freeformTags` and `definedTags`.
  `compartmentId`, `domainName`, `subscriptionEmail`, and
  `isGovernanceEnabled` remain replacement-only because the visible SDK does
  not expose truthful in-place mutation for them.
- Delete is header-only and confirm-delete-driven. `DeleteDomain` returns only
  request headers and no work-request identifier, so the runtime confirms
  deletion through `GetDomain` or fallback `ListDomains` rereads and keeps the
  finalizer until the domain is gone or reports lifecycle state `DELETED`.
- Required status projection includes the bound domain identifier, domain
  name, owner tenancy, lifecycle, raw SDK status, TXT record, timestamps, and
  tags with no secret side effects.
