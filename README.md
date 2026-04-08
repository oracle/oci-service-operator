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

The customer docs are organized for the GitHub Pages site at
[oracle.github.io/oci-service-operator](https://oracle.github.io/oci-service-operator/).
The checked-in source for that site lives under [docs/](docs/).

For the quickest repo-local path, start with:

- [Installation](docs/installation.md#installation)
- [Quick start with KRO](docs/user-guide.md)
- [Supported Resources](docs/reference/index.md)
- [API Reference](docs/reference/api/index.md)
- [Contributor Docs](docs/contributor/index.md)

## Installation

Start with the [Installation](docs/installation.md#installation) guide for OLM
prerequisites, authentication setup, and published per-package bundle commands.
Then use [Quick start with KRO](docs/user-guide.md) for the end-to-end MySQL
example that assumes those installation prerequisites are already complete. Use
[Supported Resources](docs/reference/index.md) for the generated package and
kind inventory behind the current docs set.

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

See the [docs site](https://oracle.github.io/oci-service-operator/) for the
customer-facing layout and [docs/index.md](docs/index.md) for the checked-in
source landing page.

## Samples

Samples for managing OCI Services/Resources using `oci-service-operator` can be
found in [config/samples](config/samples).

## Changes

See [CHANGELOG](CHANGELOG.md).

## Contributing

`oci-service-operator` welcomes contributions from the community. Before
submitting a pull request, review the [contribution guide](./CONTRIBUTING.md).

## Security

Please consult the [security guide](./SECURITY.md) for our responsible security
vulnerability disclosure process.

## License

Copyright (c) 2021 Oracle and/or its affiliates.

Released under the Universal Permissive License v1.0 as shown at <https://oss.oracle.com/licenses/upl/>.
