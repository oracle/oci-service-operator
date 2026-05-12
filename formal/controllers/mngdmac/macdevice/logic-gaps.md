---
schemaVersion: 1
surface: repo-authored-semantics
service: mngdmac
slug: macdevice
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `mngdmac/MacDevice` row after the
published runtime replaced the scaffold placeholder with an explicit
bind-existing plus terminate-only contract.

## Current runtime path

- `MacDevice` keeps the generated controller, service-manager shell, and
  registration wiring, but the live published behavior is owned by
  `pkg/servicemanager/mngdmac/macdevice/macdevice_runtime_client.go` rather
  than the unmodified generated baseline.
- The vendored SDK exposes `GetMacDevice`, `ListMacDevices`, and
  `TerminateMacDevice` on `MacDeviceClient`, while the service-local
  `GetWorkRequest` helper remains available on `MacOrderClient`. The reviewed
  runtime uses both clients because terminate follows a work request even
  though the resource itself has no create or update operations.
- The published contract is bind-existing and manage-existing only.
  Reconciliation requires explicit `spec.macOrderId` plus `spec.macDeviceId`,
  reads live state through `GetMacDevice`, and uses `ListMacDevices` only for
  exact-id absence confirmation when rereads fail. The runtime does not invent
  create or update behavior.
- Mutable drift is intentionally empty: `macOrderId` and `macDeviceId` remain
  identity-only and replacement-only once bound, while every other field on
  the CR is observed status projected from OCI.
- Lifecycle classification stays explicit. `CREATING` requeues as
  provisioning, `ACTIVE` settles success, `DELETING` and tracked delete work
  requeue as terminating, `NEEDS_ATTENTION` is terminal without requeue, and
  `DELETED` is treated as delete-confirmed or missing live state.
- Kubernetes delete is an explicit terminate contract, not a pretend
  `DeleteMacDevice` surface. The runtime sends `TerminateMacDevice`, tracks the
  returned work request through `GetWorkRequest`, then rereads `GetMacDevice`
  or `ListMacDevices` until the device is unambiguously gone before releasing
  the finalizer.
- Required status projection remains part of the repo-authored contract. The
  runtime mirrors the observed `MacDevice` body, including `id`,
  `compartmentId`, `macOrderId`, `serialNumber`, `ipAddress`,
  `lifecycleState`, `shape`, `timeCreated`, `timeUpdated`, `isMarkedDecom`,
  and `timeDecom`, alongside the shared OSOK status and async breadcrumbs.
