---
schemaVersion: 1
surface: repo-authored-semantics
service: email
slug: dkim
gaps: []
---

# Logic Gaps

The row remains at scaffold stage until its placeholder provider-fact import is
refreshed, but the runtime behavior is now live-verified.

## Current runtime contract

- Bind-or-create uses `emailDomainId` plus selector `name`; create, update, and
  delete call the corresponding Email SDK operations through generatedruntime.
- `description`, `freeformTags`, and `definedTags` reconcile in place.
  `emailDomainId` and selector `name` are replacement-only.
- `NEEDS_ATTENTION` is successful rather than failed: the DKIM key exists and
  OCI is waiting for external DNS publication. The controller exposes the DNS
  record material in status and does not own that external action.
- Delete remains finalizer-protected through `DELETING` until OCI confirms
  `DELETED` or the resource is unambiguously absent.

## Remaining scaffold boundary

- `formal/imports/email/dkim.json` is still a scaffold provider-fact import.
  Refresh it from a pinned Terraform provider checkout before promoting the
  manifest row beyond scaffold.
