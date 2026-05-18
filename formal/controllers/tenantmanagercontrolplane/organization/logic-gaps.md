---
schemaVersion: 1
surface: repo-authored-semantics
service: tenantmanagercontrolplane
slug: organization
gaps: []
---

# Logic Gaps

## Current runtime path

- `Organization` keeps the generated controller, service-manager shell, and
  registration wiring, but the reviewed runtime contract is finalized in
  `pkg/servicemanager/tenantmanagercontrolplane/organization/organization_runtime_client.go`.
- The pinned SDK exposes `GetOrganization`, `ListOrganizations`,
  `UpdateOrganization`, and service-local work-request reads, but it does not
  expose `CreateOrganization` or `DeleteOrganization`. The published runtime is
  therefore bind-existing plus update-only: it binds by explicit
  `spec.organizationId` or a unique `ListOrganizations` match on
  `spec.compartmentId`, and it fails instead of inventing a create path.
- Only `defaultUcmSubscriptionId` reconciles in place.
  `OrganizationTenancy`, `CreateChildTenancy`, and tenancy-termination helpers
  remain auxiliary and out of scope, so binding inputs stay replacement-only
  drift.
- Lifecycle handling is explicit: `CREATING` requeues as provisioning,
  `UPDATING` requeues as updating, `ACTIVE` settles success, `DELETING` and
  `DELETED` remain terminating observations, and `FAILED` is terminal failure.
- Update is work-request-backed. `UpdateOrganization` seeds
  `status.status.async.current.workRequestId`, reconcile resumes through
  `GetWorkRequest`, and success falls through a live `GetOrganization` reread
  because the update response does not return an `Organization` body.
- Delete is CR-local unbind only. Deleting the Kubernetes resource clears the
  finalizer immediately and never calls `DeleteOrganizationTenancy` or other
  tenancy termination APIs.
- Status projection is required and publishes the bound organization identity,
  compartment, default subscription, display and parent names, lifecycle
  fields, and timestamps with no secret side effects.
