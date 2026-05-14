---
schemaVersion: 1
surface: repo-authored-semantics
service: osubsubscription
slug: subscription
gaps:
  - category: bind-versus-create
    status: open
    stopCondition: "Close when the pinned SDK or verified provider facts expose a stable top-level SubscriptionSummary identity so the controller can bind and track one subscription without relying on a query-only unique list match."
  - category: lifecycle-classification
    status: open
    stopCondition: "Close when a pinned OSubscription contract publishes a stable top-level subscription status taxonomy that distinguishes retryable and settled states more precisely than the current blank-status observe-only heuristic."
---

# Logic Gaps

`osubsubscription/Subscription` must stay explicitly group-qualified and
distinct from the already-published `self/Subscription` and
`ons/Subscription` surfaces.

## Current runtime path

- `Subscription` keeps the generated controller, service-manager shell, and
  registration wiring, but the reviewed observe-only contract is owned by
  `pkg/servicemanager/osubsubscription/subscription/subscription_runtime_client.go`.
- The vendored SDK exposes `ListSubscriptions` only. The published kind is
  therefore query-backed and observe-only: reconcile requires
  `spec.compartmentId` plus exactly one of `spec.planNumber`,
  `spec.subscriptionId`, or `spec.buyerEmail`.
- The package-local read path passes `isCommitInfoRequired`, `sortOrder`,
  `sortBy`, `xOneGatewaySubscriptionId`, and `xOneOriginRegion` through the
  list request when set in spec. Pagination is controller-owned; the runtime
  reads all pages before selecting a candidate.
- Because `SubscriptionSummary` does not expose the request filter fields or a
  clean top-level OCID, the runtime projects each returned summary through a
  package-local read model that echoes the query selector fields used for that
  request. generatedruntime then accepts only a unique match for the exact
  selector. Zero matches and multiple matches fail instead of guessing.
- No OCI create, update, get, or delete path is published. Delete is
  Kubernetes-local cleanup only.
- Status projection remains required. The runtime mirrors the observed
  `SubscriptionSummary` body into the published status fields, keeps the raw
  `sdkStatus` string visible, and intentionally does not stamp
  `status.status.ocid` because the top-level summary does not prove a stable
  controller-owned identity.
- Lifecycle classification is intentionally narrow. A blank status string is
  treated as retryable `updating` because the read projection is incomplete;
  any non-empty status settles success while the raw `sdkStatus` remains
  visible on the CR.

## Open Gaps

- The top-level `SubscriptionSummary` still lacks a proven stable identity
  surface. The current rollout is therefore query-backed observe-only rather
  than tracked bind-existing by OCI ID.
- The pinned SDK does not publish a reviewed enum or stable catalog for the
  top-level `status` field. The current runtime keeps the raw string visible
  and uses only a minimal blank-status heuristic instead of claiming a richer
  lifecycle taxonomy.
