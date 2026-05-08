---
schemaVersion: 1
surface: repo-authored-semantics
service: self
slug: subscription
gaps:
  - category: drift-guard
    status: open
    stopCondition: "Close when the generated Subscription spec can distinguish omitted displayName from explicit clear-to-empty intent for UpdateSubscription."
---

# Logic Gaps

`self/Subscription` is the published SELF subscription rollout and must stay
group-qualified and distinct from the already-published `ons/Subscription`
surface.

## Current runtime path

- `Subscription` keeps the generated controller, service-manager shell, and
  registration wiring, but the reviewed runtime contract is owned by
  `pkg/servicemanager/self/subscription/subscription_runtime_client.go`.
- The vendored SDK exposes direct
  `Create/Get/List/Update/DeleteSubscription` operations plus
  `ChangeSubscriptionCompartment` and service-local
  `GetWorkRequest/ListWorkRequests` helpers. The published runtime uses the
  shared async tracker and resumes create, update, and delete through
  `GetWorkRequest`.
- Existing-before-create reuse is bounded. The controller skips pre-create
  reuse when either `spec.compartmentId` or `spec.displayName` is blank. When
  lookup is enabled, it lists by exact `compartmentId` plus `displayName`,
  narrows candidates by `tenantId`, `sellerId`, and `productId`, then rereads
  each candidate through `GetSubscription` and reuses only a unique match in a
  reusable lifecycle.
- Lifecycle handling is detail-aware. The runtime normalizes
  `lifecycleDetails` into the effective observed lifecycle used for shared
  async and delete-confirmation decisions because the coarse
  `lifecycleState` surface alone only reports `ACTIVE`, `INACTIVE`, `DELETED`,
  or `FAILED`. `CREATED`, `PENDING_ACTIVATION`, `PROVISIONING_STARTED`, and
  `PROVISIONING_COMPLETED` are provisioning; `UPDATING` is updating;
  `DELETING` is terminating; `ACTIVE` is success; and
  `PROVISIONING_FAILED`, `FAILED`, `EXPIRED`, and `TERMINATED` are terminal
  failures.
- Mutable drift is explicit: only `displayName`, `freeformTags`, and
  `definedTags` reconcile in place. `ChangeSubscriptionCompartment` stays out
  of scope, and `compartmentId`, `tenantId`, `sellerId`, `productId`,
  `subscriptionDetails`, `sourceType`, `additionalDetails`, `realm`, and
  `region` remain replacement-only drift. Empty-map tag clears are preserved.
- Delete confirmation is required, not best-effort. The controller keeps the
  finalizer until the delete work request reaches a terminal state and
  `GetSubscription` or list fallback confirms lifecycle `DELETED` or NotFound.
- Shared formal/docs/catalog surfaces keep `self/Subscription` explicitly
  group-qualified so it never collides with `ons/Subscription`.

## Open Gap

- `displayName` clear-to-empty intent is not representable in the current
  generated `Subscription` spec shape. Empty strings are treated as omission
  during update reconciliation, so the runtime leaves the live OCI value in
  place instead of sending an explicit clear until the spec can distinguish
  omission from clear-to-empty intent.
