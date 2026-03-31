# OCI Service Operator for Kubernetes

- [Introduction](#introduction)
- [Installation](installation.md#installation)
- [Services](services.md#services)
- [Generator Contract](api-generator-contract.md#osok-generator-contract)
- [Validator Guide](validator-guide.md#osok-validator-guide)
- [Troubleshooting](TROUBLESHOOT.md)

## Introduction

The OCI Service Operator for Kubernetes (OSOK) now ships generator-owned API,
controller, service-manager, registration, and package outputs from the checked-in
service map in `internal/generator/config/services.yaml`.

The default deployment also mounts
`config/manager/controller_manager_config.yaml` and passes
`--config=controller_manager_config.yaml` to the manager. That
`ControllerManagerConfig` file is now the authoritative source for
controller-runtime settings in the packaged deployment; unknown fields or
mismatched `apiVersion` / `kind` values fail startup during strict validation.

The legacy Autonomous Database, MySQL, and Streaming walkthroughs were removed
because they described the pre-v2 handwritten workflow rather than the current
generated-runtime contract. Start with:

- `config/samples/` for concrete manifests
- `docs/installation.md#controller-manager-config` for the default manager
  deployment contract
- `docs/services.md` for the supported API groups
- `docs/api-generator-contract.md` for ownership and regeneration rules
- `docs/validator-guide.md` for validation and regression gates
