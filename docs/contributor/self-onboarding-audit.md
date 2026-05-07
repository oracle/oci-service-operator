# SELF Service Onboarding Audit

This audit is the `US-118` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/self` before `services.yaml` publishes the
service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `self` package in the module cache; the
  repo lacked `vendor/github.com/oracle/oci-go-sdk/v65/self` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/self` so `go mod vendor` keeps the
  package in the branch-local inputs.

## SDK Audit

### `Subscription`

- Full CRUD family is present: `CreateSubscription`, `GetSubscription`,
  `ListSubscriptions`, `UpdateSubscription`, and `DeleteSubscription`.
- Additional mutator is present: `ChangeSubscriptionCompartment`.
- `CreateSubscriptionResponse` and `GetSubscriptionResponse` return
  `Subscription`.
- `UpdateSubscriptionResponse`, `DeleteSubscriptionResponse`, and
  `ChangeSubscriptionCompartmentResponse` return workrequest headers only, not
  a `Subscription` body.
- `ListSubscriptionsResponse` returns `SubscriptionCollection`.
- `ListSubscriptionsRequest` exposes `compartmentId`, `displayName`, `id`,
  `lifecycleDetails`, page, and sort controls. Notably, the list filter is on
  `lifecycleDetails`, not the coarse `lifecycleState`.
- The coarse `Subscription` lifecycle states are `ACTIVE`, `INACTIVE`,
  `DELETED`, and `FAILED`.
- Transitional and provisioning phases live in `LifecycleDetails` instead,
  including `CREATED`, `PENDING_ACTIVATION`, `PROVISIONING_STARTED`,
  `PROVISIONING_COMPLETED`, `PROVISIONING_FAILED`, `ACTIVE`, `EXPIRED`,
  `TERMINATED`, `FAILED`, `DELETING`, `UPDATING`, and `DELETED`.
- The package exposes service-local `GetWorkRequest`, `ListWorkRequests`,
  `ListWorkRequestErrors`, and `ListWorkRequestLogs` helpers.
- The package also exposes partner-integration flows such as
  `ResolveSubscription` and `ActivateSubscription`, but those are distinct from
  the main `Subscription` CRUD surface.

### Auxiliary Families

- `ResolveSubscription`, `ActivateSubscription`, and partner-subscription
  shapes should stay unpublished initially.
- `ChangeSubscriptionCompartment` should remain an explicit follow-up decision
  rather than silently joining the first published contract.

## Generator Implications For `US-126`

- `Subscription` is the planned first published kind for `US-126`.
- Recommended `formalSpec` is `subscription`.
- Recommended async classification is `workrequest`.
- `Subscription` looks viable as a controller-backed rollout because the
  service ships a full CRUD family plus workrequest helpers, and the model
  includes both steady-state lifecycle and detailed provisioning phases.
- The required risk callout is explicit here: `self/Subscription` must stay
  service-qualified and distinct from the already-published `ons/Subscription`
  in formal rows, docs, package indexes, registrations, and catalog surfaces.
  `US-126` must not overwrite or ambiguously merge the two `Subscription`
  kinds.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching `terraform-provider-oci` resource or data-source
  surfaces, or a checked-in provider-fact import, for the SELF `Subscription`
  model in the accessible local provider/docs layout.
- Because the repo already publishes `ons/Subscription`, `US-126` should treat
  provider-backed imports as absent or unconfirmed and keep all shared docs and
  metadata explicitly group-qualified.
