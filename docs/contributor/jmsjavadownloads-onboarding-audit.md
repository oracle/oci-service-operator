# JMS Java Downloads Onboarding Audit

This audit is the `US-156` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/jmsjavadownloads` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `jmsjavadownloads` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/jmsjavadownloads` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/jmsjavadownloads` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `JavaDownloadToken`

- Full CRUD family is present: `CreateJavaDownloadToken`,
  `GetJavaDownloadToken`, `ListJavaDownloadTokens`,
  `UpdateJavaDownloadToken`, and `DeleteJavaDownloadToken`.
- `CreateJavaDownloadTokenDetails` requires `displayName`, `description`,
  `compartmentId`, `timeExpires`, `javaVersion`, and `licenseType`.
- `CreateJavaDownloadTokenResponse` and `GetJavaDownloadTokenResponse` return
  `JavaDownloadToken`.
- `CreateJavaDownloadTokenResponse`, `UpdateJavaDownloadTokenResponse`, and
  `DeleteJavaDownloadTokenResponse` expose `OpcWorkRequestId`.
- `ListJavaDownloadTokensResponse` returns `JavaDownloadTokenCollection`.
- `ListJavaDownloadTokensRequest` requires `compartmentId` and exposes
  `lifecycleState`, `displayName`, `id`, `value`, `familyVersion`,
  `searchByUser`, page, and sort controls.
- Lifecycle states are `ACTIVE`, `CREATING`, `DELETED`, `DELETING`, `FAILED`,
  `NEEDS_ATTENTION`, and `UPDATING`.
- `JavaDownloadToken` also exposes `LifecycleDetails` values `EXPIRED`,
  `REVOKING`, and `REVOKED`.
- The resource body carries the token `value` itself, and the SDK comment is
  explicit that this value authorizes downloads.
- The package exposes service-local `GetWorkRequest`, `ListWorkRequests`,
  `ListWorkRequestErrors`, and `ListWorkRequestLogs` helpers. The
  work-request operation enum includes `CREATE_JAVA_DOWNLOAD_TOKEN`,
  `UPDATE_JAVA_DOWNLOAD_TOKEN`, and `DELETE_JAVA_DOWNLOAD_TOKEN`.

### Auxiliary Families

- Additional package families include `JavaDownloadReport`,
  `JavaLicenseAcceptanceRecord`, `JavaLicense`, `JavaDownloadRecord`,
  summarized download-count reporting, generated artifact download URLs, and
  cancel-work-request support.
- Those report, license, record, and download-url helper families should stay
  unpublished initially while the first `JavaDownloadToken` rollout lands.

## Generator Implications For `US-157`

- `JavaDownloadToken` is the planned first published kind for `US-157`.
- Recommended `formalSpec` is `javadownloadtoken`.
- Recommended async classification is `workrequest` with
  `workRequest.source=service-sdk` and phases `create`, `update`, and
  `delete`.
- `JavaDownloadToken` looks viable as a controller-backed rollout because
  create/get return the resource body directly, list projects lifecycle state,
  and the package ships service-local work-request follow-up for mutation
  paths.
- The main rollout risk is sensitive data handling: the token `value`
  authorizes downloads, so `US-157` must keep that field out of status,
  logs, and any default projection surface unless an explicit Secret policy is
  added.
- I did not find a repo-local published `JavaDownloadToken` kind today, but
  `US-157` should still keep `jmsjavadownloads` clearly distinct from the
  existing `jms` service and leave the adjacent report and license families
  unpublished.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching `terraform-provider-oci` resource or data-source
  surfaces, or a checked-in provider-fact import, for
  `jmsjavadownloads/JavaDownloadToken` in the accessible local provider/docs
  layout.
- `US-157` should treat provider-backed imports as absent or unconfirmed until
  a pinned provider surface is proven directly.
