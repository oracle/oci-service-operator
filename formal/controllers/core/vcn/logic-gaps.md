---
schemaVersion: 1
surface: repo-authored-semantics
service: core
slug: vcn
gaps: []
---

# Logic Gaps

## Repo-authored semantics

- Success is OCI `AVAILABLE`.
- Requeue covers OCI `PROVISIONING`, `UPDATING`, and `TERMINATING`.
- Delete confirmation requires `GetVcn` to stop finding the resource. If OCI
  reports `TERMINATED` before the resource disappears, that state is accepted
  as an observed pre-terminal step, but it is not the final delete confirmation.
- Supported in-place updates are `displayName`, `definedTags`,
  `freeformTags`, `securityAttributes`, and `isZprOnly`, matching the pinned
  `UpdateVcnDetails` SDK surface.
- The handwritten update-body builder keeps clear-to-empty intent for
  `displayName`, `definedTags`, and `freeformTags`: an empty desired
  `displayName` clears a retained OCI name, and explicit empty tag maps clear
  retained OCI tags instead of being pruned from the update request.
- Mutable drift treats an omitted observed `isZprOnly` value as equivalent to
  `false`, so non-ZPR VCNs do not churn on no-op updates when OCI omits the
  field from read responses.
- `securityAttributes` reconciles through the existing generated
  `map[string]shared.MapValue` CRD surface; widening that schema to match the
  SDK's full nested-object shape is out of scope for this VCN re-audit.
- Create-only drift stays out of scope for the first handwritten runtime:
  `compartmentId`, `dnsLabel`, IPv4 CIDR shape (`cidrBlock` and `cidrBlocks`),
  and IPv6 shape inputs (`ipv6PrivateCidrBlocks`, `isIpv6Enabled`,
  `isOracleGuaAllocationEnabled`, and `byoipv6CidrDetails`).

## Why this row is seeded

- The vendored SDK now provides enough branch-local truth to express explicit
  success, requeue, mutation, and delete-confirmation semantics for `Vcn`.
- Secret side effects and bind-by-name semantics remain out of scope because the
  generated VCN path is expected to reconcile directly against OCI identity and
  does not publish connection material.
