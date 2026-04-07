# OCI Service Operator for Kubernetes

- [Introduction](#introduction)
- [User Guide](user-guide.md)
- [Installation](installation.md#installation)
- [Supported Resources](reference/index.md#supported-resources)
- [Oracle Autonomous Database Service](adb.md#oracle-autonomous-database-service)
- [Oracle MySQL Database Service](mysql.md#oracle-mysql-database-service)
- [Oracle Streaming Service](oss.md#oracle-streaming-service)
- [Generator Contract](api-generator-contract.md#osok-generator-contract)
- [Validator Guide](validator-guide.md#osok-validator-guide)
- [Troubleshooting](TROUBLESHOOT.md)

## Introduction

The OCI Service Operator for Kubernetes (OSOK) now documents two related
distribution views:

- the default-active generated surface selected in
  `internal/generator/config/services.yaml`
- the per-package OLM bundles published from `packages/` by
  `.github/workflows/publish-service-packages.yml`

Those views overlap, but they are not identical. In particular, package names
such as `core-network` and published packages such as `apigateway` need to be
read from the package and workflow surfaces instead of inferred from
`services.yaml` alone.

> **Important:** Start with a test or non-production environment.
>
> **Do not deploy OSOK to production first.** Install and exercise the selected
> package bundle in an isolated cluster and OCI tenancy or compartment, confirm
> expected CRUD behavior, and only then promote the same flow to production.

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

- `docs/user-guide.md` as the primary end-to-end quickstart for OSOK users
- `config/samples/` for concrete manifests
- `docs/installation.md` for published bundle naming, package namespaces, and
  `v2.0.0-alpha` install and upgrade commands
- `docs/reference/index.md` for the generated package and resource catalog
- `docs/installation.md#controller-manager-config` for the default manager
  deployment contract
- `docs/adb.md`, `docs/mysql.md`, and `docs/oss.md` for the current
  service-specific guides
- `docs/api-generator-contract.md` for ownership and regeneration rules
- `docs/validator-guide.md` for validation and regression gates
