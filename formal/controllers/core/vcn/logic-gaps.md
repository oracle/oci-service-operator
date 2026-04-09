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
- Supported in-place updates are limited to `displayName`, `definedTags`, and
  `freeformTags`, matching the pinned `UpdateVcnDetails` SDK surface.
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
