---
schemaVersion: 1
surface: repo-authored-semantics
service: ocvp
slug: sddc
gaps: []
---

# Logic Gaps

## Current runtime path

- `Sddc` routes through the generated `SddcServiceManager` and
  `generatedruntime.ServiceClient`, with a small service-local wrapper in
  `pkg/servicemanager/ocvp/sddc/sddc_runtime.go` that keeps identity
  resolution explicit for create-or-bind flows.
- The generated runtime handles create, update, delete, status projection, and
  finalizer retention directly from the checked-in `ocvp/sddc`
  controller/service-manager path; there is no handwritten OCI CRUD adapter for
  this resource.
- Because `CreateSddc` returns only a work-request header and not the new SDDC
  OCID, the repo-authored runtime requires `spec.displayName` whenever no
  tracked OCI identifier is already recorded. The wrapper rejects create or
  bind attempts without `displayName` instead of guessing from broad list
  matches.

## Shared generated-runtime baseline

- Use the [shared generated-runtime baseline](../../../shared/generated-runtime-baseline.md)
  for the common bind, lifecycle, mutation, status, and delete semantics.
- Keep delete confirmation explicit with
  `finalizer_policy = retain-until-confirmed-delete`; the finalizer only clears
  after OCI reports the SDDC missing or in `DELETED`.
- No Kubernetes secret reads or secret writes are part of this resource beyond
  the ordinary spec field projection of SSH authorized keys.

## Repo-authored semantics

- Bind lookup is explicit: when `spec.displayName` is set, the runtime matches
  `ListSddcs` results on exact `compartmentId` plus `displayName` and only
  reuses a unique SDDC in reusable lifecycle states (`ACTIVE`, `CREATING`, or
  `UPDATING`). Terminal, deleting, and duplicate matches are never rebound.
- Mutable drift is limited to `displayName`, `vmwareSoftwareVersion`,
  `esxiSoftwareVersion`, `sshAuthorizedKeys`, `freeformTags`, and
  `definedTags`. Fields absent from `UpdateSddcDetails` remain create-only
  drift, including `compartmentId`, `hcxMode`, `initialConfiguration`, and
  `isSingleHostSddc`.
- OCI accepts software-version and SSH-key updates for the SDDC object, but
  those values influence future ESXi host additions rather than rewriting the
  current VMware host fleet in place. The published runtime still treats that
  API-supported surface as legitimate update behavior.
- Create and delete responses expose work-request headers, while update returns
  the SDDC body directly. The reviewed generatedruntime client records those
  breadcrumbs when present and relies on lifecycle rereads plus confirm-delete
  follow-up rather than service-local work-request polling.
