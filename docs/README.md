# OCI Service Operator for Kubernetes

- [Introduction](#introduction)
- [Installation](installation.md#installation)
- [Services](services.md#services)
- [Generator Contract](api-generator-contract.md#osok-generator-contract)
- [Parity Strategy](api-generator-parity.md#generator-parity-strategy)
- [Validator Guide](validator-guide.md#osok-validator-guide)
- [Troubleshooting](TROUBLESHOOT.md)

## Introduction

The OCI Service Operator for Kubernetes (OSOK) now ships generator-owned API,
controller, service-manager, registration, and package outputs from the checked-in
service map in `internal/generator/config/services.yaml`.

The legacy Autonomous Database, MySQL, and Streaming walkthroughs were removed
because they described the pre-v2 handwritten workflow rather than the current
generated-runtime contract. Start with:

- `config/samples/` for concrete manifests
- `docs/services.md` for the supported API groups
- `docs/api-generator-contract.md` for ownership and regeneration rules
- `docs/api-generator-parity.md` for the historical parity resources
- `docs/validator-guide.md` for validation and regression gates
