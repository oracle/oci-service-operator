# Service Manager Proxy Onboarding Audit

This audit is the `US-118` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/servicemanagerproxy` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `servicemanagerproxy` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/servicemanagerproxy` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/servicemanagerproxy` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `ServiceEnvironment`

- The package exposes `GetServiceEnvironment` and `ListServiceEnvironments`
  only; no create, update, or delete family exists.
- `GetServiceEnvironmentRequest` requires `serviceEnvironmentId` plus
  `compartmentId`; the `serviceEnvironmentId` is explicitly not an OCID.
- `GetServiceEnvironmentResponse` returns `ServiceEnvironment`.
- `ListServiceEnvironmentsResponse` returns `ServiceEnvironmentCollection`.
- `ListServiceEnvironmentsRequest` requires `compartmentId` and exposes
  `serviceEnvironmentId`, `serviceEnvironmentType`, `displayName`, page, and
  sort controls.
- The model does not expose a normal lifecycle enum. Instead it carries
  `status ServiceEntitlementRegistrationStatusEnum` with many SaaS entitlement
  transitions such as `BEGIN_ACTIVATION`, `ACTIVE`, `BEGIN_TERMINATION`,
  `TERMINATED`, `SUSPENDED`, `RELOCATED`, and several `FAILED_*` states.
- The top-level `id` and `subscriptionId` fields are explicitly not OCIDs, and
  the model is primarily a read projection around `ServiceDefinition` and
  endpoint metadata.

### Auxiliary Families

- `ServiceEnvironment` is the only discovered top-level published candidate in
  the package.
- There is no separate mutation or workrequest helper family to publish
  alongside it.

## Generator Implications For `US-122`

- `ServiceEnvironment` is the planned first published kind for `US-122`.
- Recommended `formalSpec` is `serviceenvironment`.
- Recommended async classification is `none`.
- The required non-standard risk callout is explicit here: the pinned SDK is
  get/list-only, uses non-OCID identifiers, and exposes entitlement
  registration status rather than create/update/delete semantics. `US-122`
  should publish it only as a bind-existing observe-only contract keyed by an
  explicit `serviceEnvironmentId` or tracked OCI identity; it should not
  simulate OCI mutations that the SDK does not provide.
- The wide entitlement-status enum also means the later story must define which
  states count as ready, retryable, terminal, or unsupported for an
  observe-only controller.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching `terraform-provider-oci` resource or data-source
  surfaces, or a checked-in provider-fact import, for `ServiceEnvironment` in
  the accessible local provider/docs layout.
- `US-122` should treat provider-backed imports as absent or unconfirmed until
  a pinned provider surface is proven directly.
