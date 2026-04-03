# OCI Service Operator for Kubernetes

## Introduction

The OCI Service Operator for Kubernetes (OSOK) makes it easy to create, manage, and connect to Oracle Cloud Infrastructure (OCI) resources from a Kubernetes environment. Kubernetes users can simply install OSOK and perform actions on OCI resources using the Kubernetes API removing the need to use the [OCI CLI](https://docs.oracle.com/en-us/iaas/Content/API/Concepts/cliconcepts.htm) or other [OCI developer tools](https://docs.oracle.com/en-us/iaas/Content/devtoolshome.htm) to interact with a service API.

OSOK is based on the [Operator Framework](https://operatorframework.io/), an open-source toolkit used to manage Operators. It uses the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) library, which provides high-level APIs and abstractions to write operational logic and also provides tools for scaffolding and code generation for Operators.

> **Important:** Use OSOK in a test or non-production OCI and Kubernetes
> environment first.
>
> **Do not make a production cluster your first deployment target.** Validate
> authentication, IAM policy scope, create and delete behavior, finalizers, and
> service-specific limits in an isolated test environment before promoting any
> package bundle to production.

## Start Here

The primary user-facing getting-started reference is
[docs/user-guide.md](docs/user-guide.md). It walks through an end-to-end OSOK
plus kro example that provisions an OCI MySQL DB System from one user-facing
custom resource.

**Supported API Groups**

OSOK now has two related surfaces that are easy to confuse:

- `internal/generator/config/services.yaml` is the generator source of truth and
  defines the default-active generated surface.
- `packages/` plus
  `.github/workflows/publish-service-packages.yml` define the package-local OLM
  bundles that are currently published.

The default-active generated surface in this checkout is:

- whole-service: `containerengine`, `core`, `dataflow`, `functions`, `mysql`,
  `nosql`, `psql`, `queue`, `vault`
- focused-kind: `containerinstances/ContainerInstance`,
  `database/AutonomousDatabase`, `identity/Compartment`,
  `objectstorage/Bucket`, `opensearch/OpensearchCluster`,
  `redis/RedisCluster`, `streaming/Stream`

The current subpackage publish workflow builds controller and bundle images for:
`apigateway`, `containerengine`, `containerinstances`, `core-network`,
`database`, `dataflow`, `functions`, `identity`, `mysql`, `nosql`,
`objectstorage`, `opensearch`, `psql`, `queue`, `redis`, `streaming`, and
`vault`.

A few names are intentionally package-oriented instead of matching
`services.yaml` one-for-one:

- `core-network` is a split package carved from selected `core` networking
  kinds.
- `database`, `identity`, `objectstorage`, `opensearch`, `redis`, and
  `streaming` publish focused bundles even though their default-active scope is
  narrower than the full OCI service.
- `apigateway` is published from `packages/apigateway`, even though it is not
  part of the current default-active `services.yaml` surface.

The workflow's default `subpackages=all` batch intentionally excludes `core`,
so do not assume a published `oci-service-operator-core-bundle:v2.0.0-alpha`
image unless it was released separately.

See [docs/services.md](docs/services.md#services) for the supported service map
and [config/samples](config/samples) for generated manifest examples.

## Installation

Start with the [User Guide](docs/user-guide.md) for the quickest single-resource
quickstart. Use the [Installation](docs/installation.md#installation) guide for
OLM prerequisites, authentication setup, and the published per-package bundle
commands.

## Controller Manager Config

The default `config/default` deployment turns
`config/manager/controller_manager_config.yaml` into the `manager-config`
ConfigMap and starts the manager with `--config=controller_manager_config.yaml`.
When that flag is present, the file is authoritative for controller-runtime
settings such as metrics, health probes, webhooks, cache behavior, and leader
election instead of the built-in flag defaults.

The file must keep
`apiVersion: controller-runtime.sigs.k8s.io/v1alpha1` and
`kind: ControllerManagerConfig`. OSOK now unmarshals this file strictly, so
unknown fields or mismatched type metadata fail startup instead of silently
falling back to defaults. See
[docs/installation.md](docs/installation.md#controller-manager-config) for the
deployment wiring details.

## Documentation

See the [Documentation](docs/README.md#oci-service-operator-for-kubernetes) for
the full docs index. The primary quickstart is
[docs/user-guide.md](docs/user-guide.md).

## Published Bundles

The repo still carries monolithic bundle targets in the `Makefile`, but the
current GitHub publish workflow is centered on per-package OLM bundles in GHCR.

Bundle images use:

```text
ghcr.io/<REPOSITORY_OWNER>/oci-service-operator-<GROUP>-bundle:v2.0.0-alpha
```

The matching controller images use:

```text
ghcr.io/<REPOSITORY_OWNER>/oci-service-operator-<GROUP>:v2.0.0-alpha
```

Example:

```bash
docker pull ghcr.io/<REPOSITORY_OWNER>/oci-service-operator-mysql-bundle:v2.0.0-alpha
```

See [docs/installation.md](docs/installation.md#deploy-oci-service-operator-for-kubernetes)
for install and upgrade commands and [docs/services.md](docs/services.md#published-subpackage-bundles)
for the published package list.

## Samples

Samples for managing OCI Services/Resources using `oci-service-operator`, can be found [here](config/samples).

## Changes

See [CHANGELOG](CHANGELOG.md).

## Contributing
`oci-service-operator` project welcomes contributions from the community. Before submitting a pull request, please [review our contribution guide](./CONTRIBUTING.md).

## Security

Please consult the [security guide](./SECURITY.md) for our responsible security
vulnerability disclosure process.

## License

Copyright (c) 2021 Oracle and/or its affiliates.

Released under the Universal Permissive License v1.0 as shown at <https://oss.oracle.com/licenses/upl/>.
