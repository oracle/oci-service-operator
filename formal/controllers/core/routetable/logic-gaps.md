---
schemaVersion: 1
surface: repo-authored-semantics
service: core
slug: routetable
gaps: []
---

# Logic Gaps

## Repo-authored semantics

- Success is OCI `AVAILABLE`.
- Requeue covers OCI `PROVISIONING`, `UPDATING`, and `TERMINATING`.
- Delete confirmation requires `GetRouteTable` to stop finding the resource.
  OCI `TERMINATING` and `TERMINATED` are observed intermediate states that keep
  the finalizer in place instead of confirming deletion.
- Supported in-place updates are limited to `displayName`, `definedTags`,
  `freeformTags`, and `routeRules`, matching the pinned
  `UpdateRouteTableDetails` SDK surface.
- Route-rule drift is normalized before update decisions: ordering differences
  and OCI-managed `LOCAL` rules that are not part of the desired spec do not
  trigger updates.
- Create-only drift is rejected for `compartmentId` and `vcnId`.

## Authority and scoped cleanup

- `formal/controllers/core/routetable/*` is the authoritative formal path for
  this handwritten RouteTable runtime.
- `formal/controller_manifest.tsv` still contains a separate `coreroutetable`
  row. Any ambiguity or duplication cleanup around that row is intentionally out
  of scope for this task and should be tracked separately from the RouteTable
  runtime semantics captured here.

## Why this row is seeded

- The handwritten RouteTable runtime now defines explicit success, requeue,
  mutation, and delete-confirmation semantics.
- Secret side effects and bind-by-name semantics remain out of scope because the
  RouteTable runtime reconciles directly against OCI identity and does not
  publish connection material.
