---
schemaVersion: 1
surface: repo-authored-semantics
service: objectstorage
slug: objectstoragebucket
gaps: []
---

# Logic Gaps

## Current runtime path

- `ObjectStorageBucket` is the published v2alpha controller target and now routes through the reference-style `pkg/servicemanager/objectstorage/bucket` package via `BucketServiceManager`.
- The service client delegates CRUD, list reuse, status projection, and delete confirmation to the shared `pkg/servicemanager/generatedruntime` implementation while keeping the published-kind compatibility explicit through `Kind=ObjectStorageBucket` and `SDKName=Bucket`.
- When `spec.namespace` is empty, the service client resolves the tenancy namespace with `GetNamespace` before create or delete so the published surface does not require the caller to discover that OCI path value up front.

## Repo-authored semantics

- Create or bind is explicit. Generatedruntime first checks for an existing bucket using the published `name`, resolved `namespace`, and `compartmentId`, then issues `CreateBucket` only when OCI does not already have a matching bucket.
- Successful create and update responses can be treated as synchronous success even when OCI does not return lifecycle fields. The controller still projects the live bucket response into the published status read-model and stamps `status.status.ocid`.
- Update and delete target the currently observed OCI bucket identity when one is recorded, so rename flows continue to operate on the bound bucket instead of issuing requests against only the desired post-update name.
- `storageTier` remains create-only drift, delete retains the finalizer until `DeleteBucket` succeeds and follow-up observation confirms the bucket is gone, and no Kubernetes secret reads or writes are part of this path.
