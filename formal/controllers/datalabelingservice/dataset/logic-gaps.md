---
schemaVersion: 1
surface: repo-authored-semantics
service: datalabelingservice
slug: dataset
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `datalabelingservice/Dataset` row
after the runtime audit replaced the scaffold placeholder with the reviewed
generated-runtime contract.

## Current runtime path

- `Dataset` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/datalabelingservice/dataset/dataset_runtime_client.go`.
- The reviewed runtime builds `CreateDatasetDetails` explicitly so the
  polymorphic `datasetSourceDetails` and `datasetFormatDetails` payloads reach
  OCI as concrete SDK variants with `sourceType` and `formatType`
  discriminators, while helper-only `jsonData` scaffold fields stay out of the
  request body.
- Lifecycle handling is explicit: `CREATING` requeues as provisioning,
  `UPDATING` requeues as updating, `ACTIVE` settles success, `FAILED` and
  `NEEDS_ATTENTION` are terminal failure states without requeue, and delete
  confirmation waits through `DELETING` until `DELETED` or NotFound.
- Required status projection remains part of the repo-authored contract. The
  runtime projects `status.id`, `status.compartmentId`, `status.timeCreated`,
  `status.timeUpdated`, `status.lifecycleState`, `status.annotationFormat`,
  `status.datasetSourceDetails`, `status.datasetFormatDetails`,
  `status.labelSet`, `status.displayName`, `status.description`,
  `status.lifecycleDetails`, `status.lifecycleSubstate`,
  `status.initialRecordGenerationConfiguration`,
  `status.initialImportDatasetConfiguration`, `status.labelingInstructions`,
  `status.freeformTags`, `status.definedTags`, `status.systemTags`, and
  `status.additionalProperties` when OCI returns them.

## Repo-authored semantics

- Pre-create lookup is explicit. The runtime only attempts list reuse when
  `spec.displayName` is non-empty, then queries `ListDatasets` with exact
  `compartmentId`, `annotationFormat`, `displayName`, and optional tracked `id`
  before reusing a single candidate in reusable lifecycle states. Missing
  `displayName` skips pre-create reuse instead of guessing.
- Mutation policy is explicit: only `displayName`, `description`,
  `labelingInstructions`, `freeformTags`, and `definedTags` reconcile in
  place. The handwritten update builder preserves clear-to-empty intent for
  those fields instead of dropping empty strings or empty tag maps.
- Auxiliary dataset mutators remain out of scope for the published runtime.
  `ChangeDatasetCompartment`, label-set helpers, `GenerateDatasetRecords`,
  `ImportPreAnnotatedData`, and `SnapshotDataset` are visible in provider facts
  but are not invoked by the reviewed controller-backed surface.
- Create-time inputs `compartmentId`, `annotationFormat`,
  `datasetSourceDetails`, `datasetFormatDetails`, `labelSet`,
  `initialRecordGenerationConfiguration`, and
  `initialImportDatasetConfiguration` remain explicit non-mutable drift after
  the dataset exists.
- Create and delete responses expose `opc-work-request-id`, while
  `UpdateDataset` does not. The reviewed runtime records shared request and
  async breadcrumbs when OCI returns them, but relies on lifecycle projection
  plus confirm-delete rereads instead of service-local work-request polling.
