---
schemaVersion: 1
surface: repo-authored-semantics
service: mysql
slug: mysqldbsystem
gaps:
  - category: bind-versus-create
    status: open
    stopCondition: "Formal semantics can branch between create, bind-by-id, and bind-by-display-name flows without routing through DbSystemServiceManager."
  - category: list-lookup
    status: open
    stopCondition: "Formal semantics encode the current displayName plus compartment lookup and the ACTIVE, CREATING, UPDATING, or INACTIVE lifecycle filter used before create or bind."
  - category: secret-input
    status: open
    stopCondition: "Formal semantics model the required username and password secret reads before create."
  - category: endpoint-materialization
    status: open
    stopCondition: "Formal semantics model the ACTIVE-only secret write that publishes IP address, FQDN, ports, and endpoint JSON."
  - category: status-projection
    status: open
    stopCondition: "Formal semantics either describe the handwritten OsokStatus projection or preserve it as an explicit legacy-only contract."
  - category: update-guard
    status: open
    stopCondition: "Formal semantics enumerate the limited mutable fields and keep immutable inputs from silently drifting into update requests."
  - category: delete-confirmation
    status: open
    stopCondition: "Delete is represented as an explicit unsupported path or replaced with a safe OCI delete plus confirmation flow before promotion."
  - category: legacy-adapter
    status: open
    stopCondition: "mysqldbsystem_generated_client_adapter.go is removable because the formal runtime covers the current DbSystemServiceManager behavior."
---

# Logic Gaps

## Current runtime path

- The generated `MySqlDbSystemServiceManager` is overridden by `mysqldbsystem_generated_client_adapter.go`, so create, update, and delete still run through `DbSystemServiceManager`.
- When `spec.id` is empty, the legacy manager lists DB systems by `compartmentId` and `displayName` before deciding whether to create a new system or bind to an existing one.

## Repo-authored semantics

- Create requires two Kubernetes secrets: `spec.adminUsername` must expose a `username` key and `spec.adminPassword` must expose a `password` key.
- After the DB system reaches ACTIVE, the manager writes a Kubernetes secret containing `PrivateIPAddress`, `InternalFQDN`, `AvailabilityDomain`, `FaultDomain`, `MySQLPort`, `MySQLXProtocolPort`, and serialized endpoint data.
- Status projection is manual. Only `OsokStatus` is recorded even though the OCI response carries richer state.
- Update is intentionally narrow: the handwritten logic only mutates display name, description, configuration ID, and tags.
- Delete currently returns success without issuing an OCI delete request, so finalizer removal is not confirmation.

## Why this stays on the legacy adapter

- Secret reads and endpoint secret materialization are OSOK-only semantics that do not come from provider facts.
- The bind-versus-create and update guard rules are more specific than the current generic runtime heuristics, so promotion must wait until the formal model can express them directly.
