# Data Labeling Service Onboarding Audit

This audit is the `US-84` baseline for onboarding
`github.com/oracle/oci-go-sdk/v65/datalabelingservice` before `services.yaml`
publishes the service.

## Repo Input Status

- `go.mod` remains pinned to `github.com/oracle/oci-go-sdk/v65 v65.110.0`.
- `v65.110.0` already contains the `datalabelingservice` package in the module
  cache; the repo lacked
  `vendor/github.com/oracle/oci-go-sdk/v65/datalabelingservice` only because
  nothing imported that package yet.
- `pkg/sdkimports/rollout_services.go` now blank-imports
  `github.com/oracle/oci-go-sdk/v65/datalabelingservice` so `go mod vendor`
  keeps the package in the branch-local inputs.

## SDK Audit

### `Dataset`

- Full CRUD family is present: `CreateDataset`, `GetDataset`, `ListDatasets`,
  `UpdateDataset`, and `DeleteDataset`.
- Additional mutators are present: `AddDatasetLabels`,
  `ChangeDatasetCompartment`, `GenerateDatasetRecords`,
  `RemoveDatasetLabels`, `RenameDatasetLabels`, and `SnapshotDataset`.
- `GetDatasetResponse` returns `Dataset`.
- `ListDatasetsResponse` returns `DatasetCollection`.
- `ListDatasetsRequest` exposes required `compartmentId`, plus `id`,
  `annotationFormat`, `lifecycleState`, and `displayName`, plus page and sort
  controls.
- Lifecycle states are `CREATING`, `UPDATING`, `ACTIVE`,
  `NEEDS_ATTENTION`, `DELETING`, `DELETED`, and `FAILED`.
- `CreateDatasetResponse` returns `Dataset` and exposes `OpcWorkRequestId`.
- `UpdateDatasetResponse` returns `Dataset` but does not expose
  `OpcWorkRequestId`.
- `DeleteDatasetResponse` exposes `OpcWorkRequestId`.

### Auxiliary Families

- Additional SDK-discovered families are `AnnotationFormat`, `WorkRequest`,
  `WorkRequestError`, and `WorkRequestLog`.
- `AnnotationFormat` is list-only.
- The label-management, snapshot, record-generation, and work-request
  auxiliaries should stay unpublished initially.

## Generator Implications For `US-90`

- `Dataset` is the narrowest first published kind and already matches the
  approved follow-on story.
- Recommended `formalSpec` is `dataset`.
- Recommended async classification is `lifecycle`.
- `Dataset` looks viable as a direct controller-backed generated rollout
  without handwritten runtime work if the first story stays narrow to the base
  `CreateDataset` and `UpdateDataset` field surface and leaves the label,
  snapshot, import, and record-generation helpers out of scope.
- No `observedState.sdkAliases` requirement is apparent for `Dataset`; the GET
  response already projects the selected kind directly.

## Provider-Facts Coverage

- `formal/sources.lock` pins provider facts to
  `github.com/oracle/terraform-provider-oci@eb653febb1bab4cc6650a96d404a8baf36fdf671`.
- Matching provider surfaces are `oci_data_labeling_service_dataset` as both
  the resource and singular data source, plus
  `oci_data_labeling_service_datasets` as the list data source.
- The provider resource uses `GetWorkRequest` for create, delete, and
  compartment changes, but its broader update flow also folds in package-local
  label and import helpers. The first generated rollout should therefore stay
  narrow to core `Dataset` CRUD semantics and use lifecycle polling for the
  selected published surface.
