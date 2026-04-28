---
schemaVersion: 1
surface: repo-authored-semantics
service: core
slug: securitylist
gaps: []
---

# Logic Gaps

## Repo-authored semantics

- Success is OCI `AVAILABLE`.
- Requeue covers OCI `PROVISIONING`, the defensive literal `UPDATING`,
  `TERMINATING`, and `TERMINATED`.
- Delete confirmation requires `GetSecurityList` to stop finding the resource.
  OCI `TERMINATING` and `TERMINATED` are observed intermediate states that keep
  the finalizer in place instead of confirming deletion.
- Supported in-place updates are limited to `displayName`, `definedTags`,
  `freeformTags`, `egressSecurityRules`, and `ingressSecurityRules`, matching
  the pinned `UpdateSecurityListDetails` SDK surface and the handwritten runtime.
- Ingress and egress rule comparison is order-insensitive, so semantically
  equivalent OCI rule reordering does not trigger updates.
- Create-only drift is rejected for `compartmentId` and `vcnId`.
- The runtime observes by tracked `status.osokStatus.ocid` and only recreates
  after confirmed OCI not-found clears the tracked identity.
- Status projection is authoritative for `id`, `compartmentId`, `vcnId`,
  `displayName`, tags, `egressSecurityRules`, `ingressSecurityRules`,
  `lifecycleState`, and `timeCreated`, and clears stale optional fields when OCI
  later omits them.
- Empty nested optional ICMP, TCP, UDP, and port-range objects are omitted from
  OCI request payloads and projected back as zero-value-cleared status fields
  when OCI no longer returns them.
- The runtime fails fast if the vendored `CreateSecurityListDetails`,
  `UpdateSecurityListDetails`, security-rule option structs, or rule-type enums
  drift away from the assumptions captured in this row.

## Authority and scoped cleanup

- `formal/controllers/core/securitylist/*` is the authoritative formal path for
  this handwritten SecurityList runtime.
- `formal/controller_manifest.tsv` still contains separate `coresecuritylist`
  and `coredefaultsecuritylist` scaffold rows. Any deduplication or ownership
  cleanup around those rows is intentionally out of scope for this task and is
  not folded into the `securitylist` runtime semantics here.
- List/bind-style provider datasource semantics are not part of the handwritten
  runtime contract here. The runtime observes by tracked OCID and recreates only
  on OCI not-found.

## Why this row is seeded

- The handwritten SecurityList runtime now defines explicit success, requeue,
  mutation, status-projection, and delete-confirmation semantics.
- Secret side effects and bind-by-name semantics remain out of scope because the
  SecurityList runtime reconciles directly against OCI identity and does not
  publish connection material.
