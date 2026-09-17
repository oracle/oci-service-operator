---
schemaVersion: 1
surface: repo-authored-semantics
service: filestorage
slug: export
gaps: []
---

# Logic Gaps

## Reviewed runtime contract

- `CREATING` requeues and `ACTIVE` succeeds. Export options and ID-mapping behavior update in place; export-set, file-system, and path fields identify the export.
- Delete retains the finalizer through `DELETING` until `DELETED` or confirmed absence.

## Evidence

- The checked-in recording proves create, read-only option update, and terminal deletion with real File System and Mount Target prerequisites.
- The dynamic mock also proves the initial normalization update caused by OCI materializing omitted client-option booleans as false.

No open formal gaps remain for this runtime surface.
