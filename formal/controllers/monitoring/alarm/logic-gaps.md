---
schemaVersion: 1
surface: repo-authored-semantics
service: monitoring
slug: alarm
gaps: []
---

# Logic Gaps

## Current runtime path

- `Alarm` uses the generated `AlarmServiceManager` and generated runtime client
  directly; there is no checked-in monitoring-specific OCI adapter or
  package-local runtime wrapper for this resource.
- The checked-in runtime path stays service-local under
  `pkg/servicemanager/monitoring/alarm` and `controllers/monitoring/` while the
  shared generatedruntime provides CRUD orchestration, status projection, and
  delete confirmation.
- Because the vendored Monitoring SDK surfaces `ACTIVE`, `DELETING`, and
  `DELETED` only, create and update complete through the generated
  read-after-write follow-up instead of requeueing through separate
  provisioning or updating lifecycle states.

## Shared generated-runtime baseline

- Use the [shared generated-runtime baseline](../../../shared/generated-runtime-baseline.md)
  for bind-versus-create, read-after-write observation, required status
  projection, and required delete confirmation.
- Keep delete confirmation explicit with
  `finalizer_policy = retain-until-confirmed-delete`; the finalizer only clears
  after OCI reports the Alarm missing or in `DELETED`.
- No Kubernetes secret reads or writes are part of `Alarm`.

## Repo-authored semantics

- Bind lookup is explicit: when no OCI identifier is tracked, the generated
  runtime scopes `ListAlarms` by `spec.compartmentId` plus `spec.displayName`
  and only reuses a unique Alarm in reusable lifecycle states. Duplicate
  display-name matches fail instead of guessing.
- Mutable drift is limited to `displayName`, `metricCompartmentId`,
  `metricCompartmentIdInSubtree`, `namespace`, `query`, `severity`,
  `destinations`, `isEnabled`, `resourceGroup`, `resolution`,
  `pendingDuration`, `body`, `isNotificationsPerMetricDimensionEnabled`,
  `messageFormat`, `repeatNotificationDuration`, `suppression`,
  `freeformTags`, and `definedTags`.
- `spec.compartmentId` remains replacement-only drift for the reviewed
  baseline. The Monitoring SDK exposes both `UpdateAlarmDetails.compartmentId`
  and a separate `ChangeAlarmCompartment` operation, but this row keeps alarm
  moves out of scope until a resource-local move contract is proven under
  `monitoring/alarm`.
- Create and update follow the generated read-after-write path and treat OCI
  `ACTIVE` as the steady-state success target. Delete issues `DeleteAlarm`,
  then uses `GetAlarm` or list fallback until OCI reports `DELETED` or not
  found before releasing the finalizer.

## Why this row is seeded

- `monitoring/Alarm` is the priority monitoring resource on the baseline
  generatedruntime path for this rollout, so the checked-in branch now owns an
  explicit repo-authored create, update, and delete contract for it.
