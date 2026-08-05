---
schemaVersion: 1
surface: repo-authored-semantics
service: onesubscription
slug: subscription
gaps: []
---

# Logic Gaps

No open formal gaps remain for the seeded `onesubscription/Subscription` row.
The current runtime intentionally publishes an observe-only, list-backed
contract instead of claiming a top-level OCI subscription identity that the
pinned SDK does not expose.

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

## Closed Gap

- The former `list-lookup` gap is closed by the repo-authored observe-only
  runtime contract. The pinned SDK still returns only
  `[]SubscriptionSummary`, and any identifier-looking data lives in nested
  subscribed-service records rather than on the top-level summary. The
  controller therefore keeps `status.status.ocid` empty, requires
  `compartmentId` plus exactly one query filter, accepts only one returned
  summary, and reissues the list query on each reconcile.
