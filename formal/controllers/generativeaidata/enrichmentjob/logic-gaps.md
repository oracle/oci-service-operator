---
schemaVersion: 1
surface: repo-authored-semantics
service: generativeaidata
slug: enrichmentjob
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `generativeaidata/EnrichmentJob` row
after the runtime review replaced the scaffold placeholder with the published
job-shaped contract.

## Current runtime path

- `EnrichmentJob` keeps the generated controller, service-manager shell, and
  registration wiring, but overrides the generated client seam with
  `pkg/servicemanager/generativeaidata/enrichmentjob/enrichmentjob_runtime_client.go`.
- The vendored SDK publishes `GenerateEnrichmentJob`, `GetEnrichmentJob`,
  `ListEnrichmentJobs`, and `CancelEnrichmentJob` only. The handwritten runtime
  still delegates request projection and status merge to
  `generatedruntime.ServiceClient`, but it adds create-only drift validation,
  semantic-store-scoped identity recovery, and delete-to-cancel behavior rather
  than pretending a plain CRUD surface exists.
- Required status projection mirrors the live OCI `EnrichmentJob` body on the
  CR, including `semanticStoreId`, `enrichmentJobType`,
  `enrichmentJobConfiguration`, lifecycle timestamps/details,
  `percentComplete`, and tag maps. `SUCCEEDED` is the only steady success
  lifecycle, while `FAILED` and `CANCELED` are terminal unsuccessful outcomes.

## Repo-authored semantics

- Pre-create lookup is explicit. When no OCI identifier is already tracked and
  `spec.displayName` is non-empty, the runtime issues `ListEnrichmentJobs` with
  exact `semanticStoreId`, `compartmentId`, and `displayName`, then reuses only
  a single exact display-name match. Duplicate matches fail instead of
  guessing, and empty display names skip list-before-create reuse.
- Mutation policy is explicit: `semanticStoreId` and `compartmentId` are the
  create-time lookup and identity fields, and every other published spec field
  is create-only because the pinned SDK has no `UpdateEnrichmentJob`. Once an
  OCI job exists, the runtime rejects spec drift instead of silently pretending
  in-place reconciliation is available.
- Kubernetes delete is repo-authored cancel behavior layered on top of the
  generated shell. The controller maps delete to `CancelEnrichmentJob`, retains
  the finalizer while the job remains in flight, and releases it once
  `GetEnrichmentJob` confirms `CANCELED`, `FAILED`, `SUCCEEDED`, or NotFound.
  The runtime does not claim the OCI service removes completed jobs when
  cancellation finishes.
- Identity recovery remains semantic-store scoped: `GetEnrichmentJob` and
  `CancelEnrichmentJob` both require `semanticStoreId` plus the tracked job
  OCID, so status projection keeps both values authoritative for later observe
  and delete passes.

## Import boundary

- The pinned `terraform-provider-oci` revision in `formal/sources.lock` does
  not register a `generativeaidata.EnrichmentJob` resource or datasource. The
  checked-in import for this row is therefore repo-curated from the vendored
  OCI Generative AI Data SDK request and lifecycle shape rather than refreshed
  by `formal-import`, while the runtime semantics above remain the authoritative
  contract for promotion.
