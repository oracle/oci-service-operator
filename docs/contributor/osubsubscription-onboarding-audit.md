# OSubscription Onboarding Audit

This audit is the `US-156` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/osubsubscription` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `osubsubscription` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/osubsubscription` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/osubsubscription` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `Subscription`

- The pinned `SubscriptionClient` exposes `ListSubscriptions` only; there is
  no `CreateSubscription`, `GetSubscription`, `UpdateSubscription`, or
  `DeleteSubscription` family in this package.
- `ListSubscriptions` requires `compartmentId` plus exactly one of
  `planNumber`, `subscriptionId`, or `buyerEmail`. The client comment is
  explicit that zero or multiple of those filters cause a `400` error.
- `ListSubscriptionsRequest` also exposes `isCommitInfoRequired`, page, sort
  controls, and the optional headers `x-one-gateway-subscription-id` and
  `x-one-origin-region`.
- `ListSubscriptionsResponse` returns `[]SubscriptionSummary`.
- `SubscriptionSummary` exposes `status`, `timeStart`, `timeEnd`, `currency`,
  `serviceName`, and `[]SubscribedServiceSummary`.
- The top-level summary does not expose a clean top-level OCID or stable
  tracked identifier.
- Nested `SubscribedServiceSummary` records do expose an internal `id`, but
  that identity is for the subscribed-service line item rather than the
  top-level `SubscriptionSummary`.
- The package does not expose a top-level `GetSubscription` or a nested
  `GetSubscribedService` helper that could rescue the top-level identity gap.

### Auxiliary Families

- Additional package clients cover commitments and rate cards, plus the nested
  commitment and rate-card detail families attached to subscribed services.
- Those helper families should stay unpublished initially while the first
  `Subscription` contract remains limited to a truthful query-backed surface.

## Generator Implications For `US-159`

- `Subscription` is the planned first published kind for `US-159`.
- Recommended `formalSpec` is `subscription`.
- Recommended async classification is `none`.
- `Subscription` is only viable as a query-backed observe-only or
  bind-existing-over-list contract. The pinned SDK does not offer a truthful
  create, update, delete, or direct get path for the top-level summary.
- The top-level identity risk is explicit here: any identifier-looking field
  lives only in nested subscribed-service shapes, not on
  `SubscriptionSummary` itself. `US-159` should leave that logic gap explicit
  rather than inventing tracked OCIDs or CRUD semantics.
- `x-one-gateway-subscription-id` and `x-one-origin-region` are request-level
  access knobs on the list call. If they turn out to be required for truthful
  access, `US-159` must surface them explicitly or record the gap instead of
  silently dropping them.
- The repo already publishes both `ons/Subscription` and `self/Subscription`,
  so `US-159` must keep docs, formal rows, package indexes, and registrations
  explicitly service-qualified.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching `terraform-provider-oci` resource or data-source
  surfaces, or a checked-in provider-fact import, for
  `osubsubscription/Subscription` in the accessible local provider/docs
  layout.
- `US-159` should treat provider-backed imports as absent or unconfirmed and
  keep the published surface explicitly group-qualified.
