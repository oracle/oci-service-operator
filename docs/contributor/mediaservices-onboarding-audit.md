# Media Services Onboarding Audit

This audit is the `US-118` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/mediaservices` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `mediaservices` package in the module cache;
  the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/mediaservices` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/mediaservices` so `go mod vendor` keeps
  the package in the branch-local inputs.

## SDK Audit

### `MediaAsset`

- Full CRUD family is present: `CreateMediaAsset`, `GetMediaAsset`,
  `ListMediaAssets`, `UpdateMediaAsset`, and `DeleteMediaAsset`.
- Additional mutator is present: `ChangeMediaAssetCompartment`.
- `CreateMediaAssetResponse`, `GetMediaAssetResponse`, and
  `UpdateMediaAssetResponse` return `MediaAsset`.
- `DeleteMediaAssetResponse` returns only headers; it does not return a
  `MediaAsset` body or an `opc-work-request-id`.
- `ListMediaAssetsResponse` returns `MediaAssetCollection`.
- `ListMediaAssetsRequest` exposes `compartmentId`, `displayName`,
  `lifecycleState`, `distributionChannelId`, `parentMediaAssetId`,
  `masterMediaAssetId`, `type`, `bucketName`, `objectName`,
  `mediaWorkflowJobId`, `sourceMediaWorkflowId`, and
  `sourceMediaWorkflowVersion`, plus page and sort controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`, `DELETING`,
  `DELETED`, and `FAILED`.
- `DeleteMediaAssetRequest` also exposes `deleteMode`
  (`DELETE_CHILDREN` or `DELETE_DERIVATIONS`), so delete behavior is not a
  trivial one-resource remove.
- The package does not expose service-local `GetWorkRequest`,
  `ListWorkRequests`, `ListWorkRequestErrors`, or `ListWorkRequestLogs`
  helpers, and the CRUD responses do not carry `opc-work-request-id`.

### Auxiliary Families

- Additional full CRUD families are `MediaWorkflow`,
  `MediaWorkflowConfiguration`, `MediaWorkflowJob`, `StreamCdnConfig`,
  `StreamDistributionChannel`, and `StreamPackagingConfig`.
- `MediaAssetDistributionChannelAttachment` is get/list/delete only and
  should stay unpublished initially.
- The workflow, attachment, and packaging families should stay auxiliary while
  the first asset rollout lands.

## Generator Implications For `US-119`

- `MediaAsset` is the planned first published kind for `US-119` and the
  narrowest entry point into the package.
- Recommended `formalSpec` is `mediaasset`.
- Recommended async classification is `lifecycle`.
- `MediaAsset` looks viable as a direct controller-backed generated rollout
  because get/create/update all project the resource body and list exposes
  practical lookup filters.
- The main rollout risk is delete semantics: `DELETE_CHILDREN` and
  `DELETE_DERIVATIONS`, plus parent/master asset relationships, mean
  `US-119` must make delete policy explicit instead of assuming a flat object
  lifecycle.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- I could not locate matching `terraform-provider-oci` resource or data-source
  surfaces, or a checked-in provider-fact import, for `MediaAsset` in the
  accessible local provider/docs layout.
- `US-119` should treat provider-backed imports as absent or unconfirmed until
  a pinned provider surface is proven directly.
