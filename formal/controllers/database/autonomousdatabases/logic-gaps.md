---
schemaVersion: 1
surface: repo-authored-semantics
service: database
slug: autonomousdatabases
gaps:
  - category: bind-versus-create
    status: open
    stopCondition: "Formal semantics can branch between create, bind-by-id, and bind-by-display-name flows without routing through AdbServiceManager."
  - category: list-lookup
    status: open
    stopCondition: "Formal semantics encode the current displayName plus compartment lookup and the AVAILABLE or PROVISIONING lifecycle filter used before create or bind."
  - category: secret-input
    status: open
    stopCondition: "Formal semantics distinguish admin password secret reads from Vault secretId or secretVersionNumber inputs and track the status-backed drift guard."
  - category: secret-output
    status: open
    stopCondition: "Formal semantics model optional wallet generation, default wallet naming, and the resulting Kubernetes secret contents after ACTIVE."
  - category: status-projection
    status: open
    stopCondition: "Formal semantics either describe the handwritten OsokStatus plus last-applied-secret projection or preserve it as an explicit legacy-only contract."
  - category: update-guard
    status: open
    stopCondition: "Formal semantics enumerate the updatable spec fields, additional payload passthrough, and secret-reference drift checks now encoded in needsAutonomousDatabaseUpdate and UpdateAdb."
  - category: manual-webhook
    status: open
    stopCondition: "AutonomousDatabases webhook registration is either formalized as a promotion prerequisite or proven irrelevant to the controller-backed runtime path."
  - category: delete-confirmation
    status: open
    stopCondition: "Delete is represented as an explicit unsupported path or replaced with a safe OCI delete plus confirmation flow before promotion."
  - category: legacy-adapter
    status: open
    stopCondition: "autonomousdatabases_generated_client_adapter.go is removable because the formal runtime covers the current AdbServiceManager behavior."
---

# Logic Gaps

## Current runtime path

- The generated `AutonomousDatabasesServiceManager` is overridden by `autonomousdatabases_generated_client_adapter.go`, so create, update, and delete still execute inside `AdbServiceManager`.
- The legacy manager first checks `spec.id`; when it is empty, it lists ADBs by `compartmentId` and `displayName` before deciding whether to create or bind.

## Repo-authored semantics

- Admin password handling is split: Kubernetes secret input is required when `spec.secretId` is empty, while Vault-backed updates use `spec.secretId` and `spec.secretVersionNumber` and persist the last applied values in status.
- Wallet generation is optional and only runs after the resource reaches ACTIVE; the wallet secret defaults to `<metadata.name>-wallet` when `spec.wallet.walletName` is empty.
- Status projection is intentionally narrow. The controller records `OsokStatus` and the last applied secret reference fields instead of mirroring the full OCI response payload.
- `AutonomousDatabases` still has a manual webhook registration path, even though the current webhook body is mostly placeholder logic.
- Delete currently returns success without an OCI delete request, so finalizer removal cannot be treated as delete confirmation.

## Why this stays on the legacy adapter

- `CreateAdb` and `UpdateAdb` both inject additional spec payload beyond the hand-picked fields, which is richer than the current generic runtime contract.
- The current bind-versus-create, secret drift, wallet generation, and no-op delete semantics must be preserved or blocked explicitly before formal promotion can remove the adapter.
