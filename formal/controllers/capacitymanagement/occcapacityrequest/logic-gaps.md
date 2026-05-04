---
schemaVersion: 1
surface: repo-authored-semantics
service: capacitymanagement
slug: occcapacityrequest
gaps: []
---

# Logic Gaps

No open logic gaps remain for the seeded
`capacitymanagement/OccCapacityRequest` row after the runtime review replaced
the scaffold placeholder with the published lifecycle-backed contract.

## Current runtime path

- `OccCapacityRequest` keeps the generated controller, service-manager shell,
  and registration wiring, but the live runtime contract is owned by
  `pkg/servicemanager/capacitymanagement/occcapacityrequest/occcapacityrequest_runtime_client.go`
  rather than the generated baseline in
  `pkg/servicemanager/capacitymanagement/occcapacityrequest/occcapacityrequest_serviceclient.go`.
- The vendored SDK exposes direct
  `Create/Get/List/Update/DeleteOccCapacityRequest` operations. The public list
  surface is `ListOccCapacityRequests`; the SDK also exposes
  `ListOccCapacityRequestsInternal` and `UpdateInternalOccCapacityRequest`, but
  those internal helpers remain out of scope for the published runtime because
  they require `occCustomerGroupId`, which the CR does not model as part of
  controller-owned identity.
- Lifecycle handling is explicit: `CREATING` requeues as provisioning,
  `UPDATING` requeues as updating, `ACTIVE` settles success, `DELETING`
  requeues as terminating, `DELETED` is a delete-confirmation target, and
  `FAILED` is terminal without requeue. Create, update, and delete all surface
  `retry-after` headers, and the reviewed runtime uses those hints to shorten
  or lengthen requeue cadence instead of always falling back to the generic
  one-minute polling interval.
- Bind resolution is intentionally strict. The runtime never adopts an
  untracked live capacity request from either public or internal list APIs.
  `displayName` is mutable, the public list surface cannot prove full
  create-time equivalence for fields such as `region`,
  `dateExpectedCapacityHandover`, and `details`, and the internal list requires
  extra customer-group identity the CR does not own. The controller therefore
  binds only recorded OCI identity and creates a new request when no OCI ID is
  already tracked.
- Mutable drift is limited to the public update body:
  `displayName`, `requestState`, `freeformTags`, and `definedTags` reconcile in
  place. The handwritten update builder preserves clear-to-empty intent for
  both tag maps and for `displayName` when the spec intentionally clears it.
  All other meaningful spec fields remain create-time only drift for the
  published runtime.
- Required status projection remains part of the repo-authored contract. The
  runtime projects OSOK lifecycle conditions, shared async breadcrumbs, and the
  published `status.id`, `status.compartmentId`,
  `status.occAvailabilityCatalogId`, `status.displayName`, `status.namespace`,
  `status.occCustomerGroupId`, `status.region`,
  `status.availabilityDomain`, `status.dateExpectedCapacityHandover`,
  `status.requestState`, `status.timeCreated`, `status.timeUpdated`,
  `status.lifecycleState`, `status.details`, `status.description`,
  `status.requestType`, `status.lifecycleDetails`, `status.freeformTags`,
  `status.definedTags`, and `status.systemTags` fields when OCI returns them.
- Delete confirmation is required, not best-effort. The finalizer stays until
  `DeleteOccCapacityRequest` succeeds and `GetOccCapacityRequest` confirms the
  resource is gone or reports lifecycle state `DELETED`.
