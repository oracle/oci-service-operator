# OCI Service Operator for Kubernetes

## Introduction

The OCI Service Operator for Kubernetes (OSOK) makes it easy to create, manage, and connect to Oracle Cloud Infrastructure (OCI) resources from a Kubernetes environment. Kubernetes users can simply install OSOK and perform actions on OCI resources using the Kubernetes API removing the need to use the [OCI CLI](https://docs.oracle.com/en-us/iaas/Content/API/Concepts/cliconcepts.htm) or other [OCI developer tools](https://docs.oracle.com/en-us/iaas/Content/devtoolshome.htm) to interact with a service API.

OSOK is based on the [Operator Framework](https://operatorframework.io/), an open-source toolkit used to manage Operators. It uses the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) library, which provides high-level APIs and abstractions to write operational logic and also provides tools for scaffolding and code generation for Operators.

**Supported API Groups**

OSOK now ships generator-owned APIs and runtime scaffolding from the checked-in
service map in `internal/generator/config/services.yaml`. The current inventory
spans `artifacts`, `certificates`, `certificatesmanagement`,
`containerengine`, `core`, `database`, `dns`, `events`, `functions`,
`identity`, `keymanagement`, `limits`, `loadbalancer`, `logging`,
`monitoring`, `mysql`, `networkloadbalancer`, `nosql`, `objectstorage`,
`ons`, `psql`, `queue`, `secrets`, `streaming`, `vault`, and
`workrequests`.

See [docs/services.md](docs/services.md#services) for the supported service map
and [config/samples](config/samples) for generated manifest examples.

## Installation

See the [Installation](docs/installation.md#install-operator-sdk) instructions for detailed installation and configuration of OCI Service Operator for Kubernetes.

## Documentation

See the [Documentation](docs/README.md#oci-service-operator-for-kubernetes) for complete details on installation, security and service related configurations of OCI Service Operator for Kubernetes.

## Release Bundle

The OCI Service Operator for Kubernetes is packaged as Operator Lifecycle Manager (OLM) Bundle for making it easy to install in Kubernetes Clusters. The bundle can be downloaded as docker image using below command.

```
docker pull iad.ocir.io/oracle/oci-service-operator-bundle:<VERSION>
```

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
