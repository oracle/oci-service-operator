# ZPR Onboarding Audit

This audit is the `US-118` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/zpr` before `services.yaml` publishes the
service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `zpr` package in the module cache; the repo
  lacked `vendor/github.com/oracle/oci-go-sdk/v65/zpr` only because nothing
  imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/zpr` so `go mod vendor` keeps the package
  in the branch-local inputs.

## SDK Audit

### `ZprPolicy`

- Full CRUD family is present: `CreateZprPolicy`, `GetZprPolicy`,
  `ListZprPolicies`, `UpdateZprPolicy`, and `DeleteZprPolicy`.
- `CreateZprPolicyResponse` returns `ZprPolicy` and `OpcWorkRequestId`.
- `GetZprPolicyResponse` returns `ZprPolicy`.
- `ListZprPoliciesResponse` returns `ZprPolicyCollection`.
- `ListZprPoliciesRequest` exposes `compartmentId`, `lifecycleState`, `name`,
  `id`, page, and sort controls.
- Lifecycle states are `ACTIVE`, `CREATING`, `UPDATING`, `DELETING`,
  `DELETED`, `FAILED`, and `NEEDS_ATTENTION`.
- `UpdateZprPolicyResponse` and `DeleteZprPolicyResponse` return workrequest
  headers only, not a `ZprPolicy` body.
- The package exposes a policy-specific workrequest seam:
  `GetZprPolicyWorkRequest`, `ListZprPolicyWorkRequests`,
  `ListZprPolicyWorkRequestErrors`, and `ListZprPolicyWorkRequestLogs`.

### Auxiliary Families

- `Configuration` is a second top-level family in the package and should stay
  unpublished initially.
- Policy workrequest helper families are auxiliary support surfaces rather than
  separate published kinds.

## Generator Implications For `US-124`

- `ZprPolicy` is the planned first published kind for `US-124`.
- Recommended `formalSpec` is `zprpolicy`.
- Recommended async classification is `workrequest`.
- `ZprPolicy` looks viable as a controller-backed rollout because the resource
  has normal get/list lifecycle projection and the service provides both
  `opc-work-request-id` on mutations and a policy-specific workrequest helper
  seam.
- The later story still needs a deliberate choice about how much mutation
  follow-up uses workrequest polling versus direct lifecycle read-after-write,
  but the pinned SDK does prove the workrequest path exists for policies.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching `terraform-provider-oci` resource or data-source
  surfaces, or a checked-in provider-fact import, for `ZprPolicy` in the
  accessible local provider/docs layout.
- `US-124` should treat provider-backed imports as absent or unconfirmed until
  a pinned provider surface is proven directly.
