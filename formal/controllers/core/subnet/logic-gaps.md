---
schemaVersion: 1
surface: repo-authored-semantics
service: core
slug: subnet
gaps: []
---

# Logic Gaps

## Repo-authored semantics

- Success is OCI `AVAILABLE`.
- Requeue covers OCI `PROVISIONING`, `UPDATING`, and `TERMINATING`.
- Delete confirmation requires `GetSubnet` to stop finding the resource. If OCI
  reports `TERMINATED` before the resource disappears, that state is accepted
  as an observed pre-terminal step, but it is not the final delete confirmation.
- Supported in-place updates are limited to `cidrBlock`, `definedTags`,
  `dhcpOptionsId`, `displayName`, `freeformTags`, `ipv6CidrBlock`,
  `ipv6CidrBlocks`, `routeTableId`, and `securityListIds`, matching the pinned
  `UpdateSubnetDetails` SDK surface and the handwritten runtime.
- Create-only drift is rejected for `availabilityDomain`, `compartmentId`,
  `dnsLabel`, `prohibitInternetIngress`, `prohibitPublicIpOnVnic`, and `vcnId`.
- When either `spec.prohibitInternetIngress` or
  `spec.prohibitPublicIpOnVnic` is the lone requested private-subnet flag, the
  runtime accepts a post-create OCI read that projects both flags as `true`
  instead of treating the paired flag as unsupported create-only drift.
- Secret side effects are out of scope because subnet reconciliation does not
  publish connection material.

## Why this row is seeded

- The vendored SDK and handwritten runtime now provide enough branch-local truth
  to express explicit success, requeue, mutation, and delete-confirmation
  semantics for `Subnet`.
- Bind-by-name semantics remain out of scope because the subnet runtime
  reconciles directly against tracked OCI identity.
