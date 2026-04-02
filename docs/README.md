# OCI Service Operator for Kubernetes

- [Introduction](#introduction)
- [Installation](installation.md#installation)
- [Services](services.md#services)
- [Oracle Autonomous Database Service](adb.md#oracle-autonomous-database-service)
- [Oracle MySQL Database Service](mysql.md#oracle-mysql-database-service)
- [Oracle Streaming Service](oss.md#oracle-streaming-service)
- [Generator Contract](api-generator-contract.md#osok-generator-contract)
- [Validator Guide](validator-guide.md#osok-validator-guide)
- [Troubleshooting](TROUBLESHOOT.md)

## Introduction

The OCI Service Operator for Kubernetes (OSOK) now ships generator-owned API,
controller, service-manager, registration, and package outputs for the
first-wave default-active services selected in
`internal/generator/config/services.yaml`. The same config also keeps broader
backlog services available for explicit `--service <name>` generation when
local rollout work needs them.

The default deployment also mounts
`config/manager/controller_manager_config.yaml` and passes
`--config=controller_manager_config.yaml` to the manager. That
`ControllerManagerConfig` file is now the authoritative source for
controller-runtime settings in the packaged deployment; unknown fields or
mismatched `apiVersion` / `kind` values fail startup during strict validation.

The current v2 service walkthroughs cover Autonomous Database, MySQL DB
System, and Streaming flows. The pre-v2 manual compatibility guides were
removed because they no longer match the generated-runtime contract. Start
with:

- `config/samples/` for concrete manifests
- `docs/installation.md#controller-manager-config` for the default manager
  deployment contract
- `docs/services.md` for the supported API groups
- `docs/adb.md`, `docs/mysql.md`, and `docs/oss.md` for the current
  service-specific guides
- `docs/api-generator-contract.md` for ownership and regeneration rules
- `docs/validator-guide.md` for validation and regression gates
