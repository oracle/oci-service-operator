---
schemaVersion: 1
surface: repo-authored-semantics
service: onesubscription
slug: subscription
gaps:
  - category: list-lookup
    status: open
    stopCondition: "Close when the pinned OneSubscription SDK or verified provider facts expose a stable top-level Subscription identity or reread path so the runtime can persist a truthful tracked identifier instead of reissuing the list query every reconcile."
---

# Logic Gaps

## Current runtime path

- `Subscription` keeps the generated controller, service-manager shell, and
  registration wiring, but the published observe-only contract is finalized in
  `pkg/servicemanager/onesubscription/subscription/subscription_runtime_client.go`.
- The pinned SDK exposes `ListSubscriptions` only. Reconcile issues that list
  call with `compartmentId` plus exactly one of `planNumber`,
  `subscriptionId`, or `buyerEmail`, follows every `opc-next-page` token, and
  fails when the query returns zero or multiple summaries instead of guessing.
- `SubscriptionSummary` does not expose a stable top-level OCID or reread path.
  The runtime therefore does not invent a synthetic tracked identity, leaves
  `status.status.ocid` empty, and reissues the query on each observe pass.
- Delete is CR-local unbind only. Removing the Kubernetes object releases
  control immediately and never calls an OCI delete helper because the pinned
  SDK exposes none.

## Repo-authored semantics

- Status projection is required. The runtime publishes the raw
  `SubscriptionSummary.status` value as `status.sdkStatus` alongside
  `timeStart`, `timeEnd`, `currency`, `serviceName`, `holdReason`,
  `timeHoldReleaseEta`, and nested `subscribedServices`.
- Lifecycle classification is intentionally narrow. The top-level subscription
  `status` is a business-state string rather than a typed OCI lifecycle enum,
  so the runtime only requeues clearly transitional create-like and update-like
  tokens, treats `FAIL*` and `ERROR*` values as terminal failure, and settles
  all other observed values as success while keeping the raw `sdkStatus`
  visible on the CR.
- Mutation policy is query-only. `compartmentId` plus one of `planNumber`,
  `subscriptionId`, or `buyerEmail` define the read contract, while
  `isCommitInfoRequired` only widens the projected payload. Spec changes
  reissue the list query; the controller does not claim any in-place OCI
  update path.

## Open Gap

- Top-level identity remains an explicit `list-lookup` gap. The pinned SDK
  returns only `[]SubscriptionSummary`, and any identifier-looking data lives
  in nested subscribed-service records rather than on the top-level summary.
  The current rollout therefore cannot prove a truthful tracked identity for
  `onesubscription/Subscription`.
