# Tenant Manager Control Plane Onboarding Audit

This audit is the `US-118` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/tenantmanagercontrolplane` before
`services.yaml` publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `tenantmanagercontrolplane` package in the
  module cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/tenantmanagercontrolplane` only
  because nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/tenantmanagercontrolplane` so
  `go mod vendor` keeps the package in the branch-local inputs.

## SDK Audit

### `Organization`

- The real `Organization` model exposes `GetOrganization`,
  `ListOrganizations`, and `UpdateOrganization`; the pinned SDK does not expose
  `CreateOrganization` or `DeleteOrganization`.
- `GetOrganizationResponse` returns `Organization`.
- `ListOrganizationsResponse` returns `OrganizationCollection`.
- `ListOrganizationsRequest` requires `compartmentId`, plus page and limit
  controls.
- `UpdateOrganizationResponse` returns only `OpcWorkRequestId`; it does not
  return an `Organization` body.
- `Organization` lifecycle states are `CREATING`, `ACTIVE`, `UPDATING`,
  `DELETING`, `DELETED`, and `FAILED`.
- `UpdateOrganizationDetails` currently exposes only one mutable field:
  `defaultUcmSubscriptionId`.
- The broader `OrganizationClient` also exposes `CreateChildTenancy`,
  `GetOrganizationTenancy`, and `DeleteOrganizationTenancy`, which are related
  but not equivalent to create/delete operations on the `Organization` model
  itself.
- The package also exposes a generic `WorkRequestClient`, but the published
  `Organization` surface is still observe/update-only.

### Auxiliary Families

- `OrganizationTenancy` workflows are auxiliary and should stay unpublished
  initially.
- Other families such as `Domain`, `DomainGovernance`, `SenderInvitation`,
  `RecipientInvitation`, and subscription-mapping flows should remain out of
  scope for the first `Organization` rollout.

## Generator Implications For `US-125`

- `Organization` is the planned first published kind for `US-125`.
- Recommended `formalSpec` is `organization`.
- Recommended async classification is `workrequest` for the update path only.
- The required non-standard risk callout is explicit here: the SDK does not
  provide direct create/delete APIs for the `Organization` model, and the only
  proven mutation path is `UpdateOrganization` on `defaultUcmSubscriptionId`.
  `US-125` should therefore publish `Organization` only as a bind-existing
  plus update-only contract over the real model, not as a fake CRUD wrapper or
  a renamed `OrganizationTenancy`.
- The later story must keep `CreateChildTenancy` and
  `DeleteOrganizationTenancy` auxiliary unless it intentionally widens the
  published contract in a different story.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching `terraform-provider-oci` resource or data-source
  surfaces, or a checked-in provider-fact import, for `Organization` in the
  accessible local provider/docs layout.
- `US-125` should treat provider-backed imports as absent or unconfirmed until
  a pinned provider surface is proven directly.
