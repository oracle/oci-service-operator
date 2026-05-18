---
schemaVersion: 1
surface: repo-authored-semantics
service: oce
slug: oceinstance
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded `oce/OceInstance` row after the
runtime review replaced the scaffold semantics with the published
work-request-backed contract.

## Current runtime path

- `OceInstance` keeps the generated controller, service-manager shell, and
  registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/oce/oceinstance/oceinstance_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/oce/oceinstance/oceinstance_serviceclient.go`.
- Create, update, and delete are work-request-backed. The runtime stores the
  in-flight OCI work request in `status.async.current`, normalizes OCE
  `WorkRequestStatus*` values (`ACCEPTED`, `IN_PROGRESS`, `CANCELING`,
  `SUCCEEDED`, `FAILED`, and `CANCELED`) into shared async classes, maps
  `CREATE_OCE_INSTANCE`, `UPDATE_OCE_INSTANCE`, and `DELETE_OCE_INSTANCE`
  into create/update/delete phases, and resumes reconciliation from that
  shared async tracker across requeues.
- Create-time identity recovery is work-request-backed. The runtime records the
  tracked OceInstance OCID when OCE exposes it and otherwise resolves the
  created resource OCID from work-request resources before reading the
  OceInstance by ID and projecting status.
- Bind resolution is bounded. When no OCI identifier is tracked, the runtime
  only attempts pre-create reuse when `spec.name` is non-empty and then adopts
  only a unique `ListOceInstances` match on exact `compartmentId`,
  `tenancyId`, and `name` via the list API's `displayName` filter. Summaries
  in `FAILED`, `DELETING`, or `DELETED` are not reused, and duplicate
  exact-name matches fail instead of binding arbitrarily.
- Mutable drift is limited to `addOnFeatures`, `definedTags`, `description`,
  `drRegion`, `freeformTags`, `instanceLicenseType`, `instanceUsageType`, and
  `wafPrimaryDomain`. The handwritten update-body builder preserves
  clear-to-empty intent for the supported string, slice, and tag fields, while
  `compartmentId`, `adminEmail`, `identityStripe`, `instanceAccessType`,
  `name`, `objectStorageNamespace`, `tenancyId`, `tenancyName`, and
  `upgradeSchedule` remain replacement-only drift. `IdcsAccessToken` remains a
  create-time input that OCI does not echo back after create, so post-create
  parity ignores it rather than trying to mutate it in place. The provider-only
  `ChangeOceInstanceCompartment` auxiliary operation stays out-of-scope for the
  published runtime.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, the shared async tracker, and the
  published `status.id`, `status.guid`, `status.compartmentId`, `status.name`,
  `status.tenancyId`, `status.idcsTenancy`, `status.tenancyName`,
  `status.objectStorageNamespace`, `status.adminEmail`, `status.description`,
  `status.identityStripe`, `status.instanceUsageType`,
  `status.addOnFeatures`, `status.upgradeSchedule`,
  `status.wafPrimaryDomain`, `status.instanceAccessType`,
  `status.instanceLicenseType`, `status.timeCreated`,
  `status.timeUpdated`, `status.lifecycleState`,
  `status.lifecycleDetails`, `status.drRegion`, `status.stateMessage`,
  `status.freeformTags`, `status.definedTags`, `status.systemTags`, and
  `status.service` read-model fields when OCE returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  the delete work request is terminal and `GetOceInstance` or fallback
  `ListOceInstances` confirms the OceInstance is gone or reports
  lifecycle state `DELETED`.
