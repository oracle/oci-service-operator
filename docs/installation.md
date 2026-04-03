# Installation

> **Important:** Use this guide in a test or non-production environment first.
>
> **Do not treat your first OSOK install as a production rollout.** Validate
> credentials, IAM policies, bundle installation, reconciliation behavior, and
> cleanup paths in an isolated cluster before deploying the same package to
> production.

* [Pre-Requisites](#pre-requisites)
* [Install Operator SDK](#install-operator-sdk)
* [Install Operator Lifecycle Manager (OLM)](#install-olm)
* [Deploy OCI Service Operator for Kubernetes](#deploy-oci-service-operator-for-kubernetes)

## Pre-Requisites

* Kubernetes Cluster
* [Operator SDK](https://sdk.operatorframework.io/)
* [Operator Lifecycle Manager (OLM)](https://olm.operatorframework.io/docs/getting-started/)
* `kubectl` to control the Kubernetes Cluster. Please make sure it points to the above Kubernetes Cluster.

## Install Operator SDK

The Operator SDK installation is documented in detail by the operator-sdk project. Please follow the document [here](https://sdk.operatorframework.io/docs/installation/) to install it.

## Install Operator Lifecycle Manager (OLM)

### Install OLM

Install the OLM from the operator-sdk, you can use the following command:
```bash
$ operator-sdk olm install --version 0.20.0
...
...
INFO[0079] Successfully installed OLM version "latest"
```

### Verify Installation

You can verify your installation of OLM by first checking for all the necessary CRDs in the cluster:

```bash
$ operator-sdk olm status
```

Output of the above command
```bash
INFO[0007] Fetching CRDs for version "0.20.0"
INFO[0007] Fetching resources for resolved version "v0.20.0"
INFO[0031] Successfully got OLM status for version "0.20.0"

NAME                                            NAMESPACE    KIND                        STATUS
operatorgroups.operators.coreos.com                          CustomResourceDefinition    Installed
operatorconditions.operators.coreos.com                      CustomResourceDefinition    Installed
olmconfigs.operators.coreos.com                              CustomResourceDefinition    Installed
installplans.operators.coreos.com                            CustomResourceDefinition    Installed
clusterserviceversions.operators.coreos.com                  CustomResourceDefinition    Installed
olm-operator-binding-olm                                     ClusterRoleBinding          Installed
operatorhubio-catalog                           olm          CatalogSource               Installed
olm-operators                                   olm          OperatorGroup               Installed
aggregate-olm-view                                           ClusterRole                 Installed
catalog-operator                                olm          Deployment                  Installed
cluster                                                      OLMConfig                   Installed
operators.operators.coreos.com                               CustomResourceDefinition    Installed
olm-operator                                    olm          Deployment                  Installed
subscriptions.operators.coreos.com                           CustomResourceDefinition    Installed
aggregate-olm-edit                                           ClusterRole                 Installed
olm                                                          Namespace                   Installed
global-operators                                operators    OperatorGroup               Installed
operators                                                    Namespace                   Installed
packageserver                                   olm          ClusterServiceVersion       Installed
olm-operator-serviceaccount                     olm          ServiceAccount              Installed
catalogsources.operators.coreos.com                          CustomResourceDefinition    Installed
system:controller:operator-lifecycle-manager                 ClusterRole                 Installed
```

## Deploy OCI Service Operator for Kubernetes

> **Production caution:** Run the selected bundle in a test environment first
> and verify create, update, and delete behavior before using it in a
> production cluster.

### Enable Instance Principal

The OCI Service Operator for Kubernetes needs OCI Instance Principal details to provision and manage OCI services/resources in the customer tenancy. This is the recommended approach for running OSOK within OCI.

The customer is required to create a OCI dynamic group as detailed [here](https://docs.oracle.com/en-us/iaas/Content/Identity/Tasks/managingdynamicgroups.htm#Managing_Dynamic_Groups).

Once the dynamic group is created, below sample matching rule can be added to the dynamic group
```
#### Below rule matches the kubernetes worker instance ocid or the compartment where the worker instances are running

Any {instance.id = 'ocid1.instance.oc1.iad..exampleuniqueid1', instance.compartment.id = 'ocid1.compartment.oc1..exampleuniqueid2'}

```

Customer needs to create an OCI Policy that can be tenancy wide or in the compartment for the dynamic group created above.

```
### Tenancy based OCI Policy for the dynamic group
Allow dynamic-group <DYNAMICGROUP_NAME> to manage <OCI_SERVICE_1> in tenancy
Allow dynamic-group <DYNAMICGROUP_NAME> to manage <OCI_SERVICE_2> in tenancy
Allow dynamic-group <DYNAMICGROUP_NAME> to manage <OCI_SERVICE_3> in tenancy
Allow dynamic-group <DYNAMICGROUP_NAME> to manage <OCI_SERVICE_4> in tenancy

### Compartment based OCI Policy for the dynamic group
Allow dynamic-group <DYNAMICGROUP_NAME> to manage <OCI_SERVICE_1> in compartment <NAME_OF_THE_COMPARTMENT>
Allow dynamic-group <DYNAMICGROUP_NAME> to manage <OCI_SERVICE_2> in compartment <NAME_OF_THE_COMPARTMENT>
Allow dynamic-group <DYNAMICGROUP_NAME> to manage <OCI_SERVICE_3> in compartment <NAME_OF_THE_COMPARTMENT>
Allow dynamic-group <DYNAMICGROUP_NAME> to manage <OCI_SERVICE_4> in compartment <NAME_OF_THE_COMPARTMENT>
```
Note: the <OCI_SERVICE_1>, <OCI_SERVICE_2> represents in the OCI Services like "autonomous-database-family", "instance_family", etc.

### Enable User Principal

The OCI Service Operator for Kubernetes needs OCI user credentials details to provision and manage OCI services/resources in the customer tenancy. This approach is recommended when OSOK is deployed outside OCI.

The users required to create a Kubernetes secret as detailed below.

The controller reads `ocicredentials` from its own namespace. For the
published per-package bundles, that namespace is normally
`oci-service-operator-<GROUP>-system`. For the legacy monolithic install, it is
`oci-service-operator-system`.

If you want to create the secret before installing the bundle, create the
operator namespace first. If you install the bundle first, the namespace is
created by the package manifests and you can create the secret afterward.

Create a yaml file using below details
```yaml
apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: <OPERATOR_NAMESPACE>
```

Create the namespace in the kubernetes cluster using below command
```bash
$ kubectl apply -f <FILE_NAME_ABOVE>
```

The secret should have the below Keys and respective values for it:

| Key | Description |
| --------- | ----------- |
| `tenancy` | The OCID of your tenancy |
| `fingerprint`    | The Fingerprint of your OCI user |
| `user`    | OCID of the user |
| `privatekey`    | The OCI User private key |
| `passphrase`    | The passphrase of the private key. This is mandatory and if the private key does not have a passphrase, then set the value to an empty string. |
| `region`    | The region in which the OKE cluster is running. The value should be in OCI region format. Example: us-ashburn-1 |

Run the below command to create the secret named `ocicredentials`. Replace the
values with your user credentials.

```bash
$ kubectl -n <OPERATOR_NAMESPACE> create secret generic ocicredentials \
--from-literal=tenancy=<CUSTOMER_TENANCY_OCID> \
--from-literal=user=<USER_OCID> \
--from-literal=fingerprint=<USER_PUBLIC_API_KEY_FINGERPRINT> \
--from-literal=region=<USER_OCI_REGION> \
--from-literal=passphrase=<PASSPHRASE_STRING> \
--from-file=privatekey=<PATH_OF_USER_PRIVATE_API_KEY>
```

The controller deployment looks for a secret named `ocicredentials` by default.
Create that secret in the operator's own namespace, for example
`oci-service-operator-mysql-system` for the MySQL bundle.

The customer should create a OSOK operator user and can add him to a IAM group `osok-operator-group`. Customer should create an OCI Policy that can be tenancy wide or in the compartment to manage the OCI Services

```
### Tenancy based OCI Policy for user
Allow group <OSOK_OPERATOR_GROUP> to manage <OCI_SERVICE_1> in tenancy
Allow group <OSOK_OPERATOR_GROUP> to manage <OCI_SERVICE_2> in tenancy
Allow group <OSOK_OPERATOR_GROUP> to manage <OCI_SERVICE_3> in tenancy
Allow group <OSOK_OPERATOR_GROUP> to manage <OCI_SERVICE_4> in tenancy

### Compartment based OCI Policy for user
Allow group <OSOK_OPERATOR_GROUP> to manage <OCI_SERVICE_1> in compartment <NAME_OF_THE_COMPARTMENT>
Allow group <OSOK_OPERATOR_GROUP> to manage <OCI_SERVICE_2> in compartment <NAME_OF_THE_COMPARTMENT>
Allow group <OSOK_OPERATOR_GROUP> to manage <OCI_SERVICE_3> in compartment <NAME_OF_THE_COMPARTMENT>
Allow group <OSOK_OPERATOR_GROUP> to manage <OCI_SERVICE_4> in compartment <NAME_OF_THE_COMPARTMENT>
```
Note: the <OCI_SERVICE_1>, <OCI_SERVICE_2> represents in the OCI Services like "autonomous-database-family", "instance_family", etc.

### Enable Security Token

OSOK also supports OCI security-token authentication for deployments outside OCI.
This mode uses the OCI SDK session-token provider, so the manager pod must read a
config file, private key, and security token from mounted files.

When `auth_type=security_token` is present in the `ocicredentials` secret, the
manager mounts that secret at `/etc/oci` and loads the OCI config from
`/etc/oci/config` by default. You can override the config path with the optional
secret key `config_file_path`, and override the OCI profile with the optional
secret key `config_file_profile` (default: `DEFAULT`).

Create a config file whose paths match the files inside the manager pod. A
working example is:

```ini
[DEFAULT]
tenancy=ocid1.tenancy.oc1..<example>
region=us-ashburn-1
fingerprint=<USER_PUBLIC_API_KEY_FINGERPRINT>
key_file=/etc/oci/privatekey
security_token_file=/etc/oci/security_token
```

Create the `ocicredentials` secret with the config, private key, and security
token files:

```bash
$ kubectl -n <OPERATOR_NAMESPACE> create secret generic ocicredentials \
--from-literal=auth_type=security_token \
--from-literal=config_file_profile=DEFAULT \
--from-file=config=<PATH_TO_OCI_CONFIG_FILE> \
--from-file=privatekey=<PATH_TO_USER_PRIVATE_API_KEY> \
--from-file=security_token=<PATH_TO_SECURITY_TOKEN_FILE> \
--from-literal=passphrase=<PASSPHRASE_STRING>
```

The `config` file stored in the secret must reference the in-pod paths
(`/etc/oci/privatekey` and `/etc/oci/security_token`), not local workstation
paths such as `~/.oci/...`.

### Published Service Bundles

The repo still supports monolithic OLM targets in the `Makefile`, but the
current GitHub workflow
`.github/workflows/publish-service-packages.yml` publishes per-package
controller images and per-package OLM bundle images to GHCR.

The published `v2.0.0-alpha` bundle naming pattern is:

```text
ghcr.io/<REPOSITORY_OWNER>/oci-service-operator-<GROUP>-bundle:v2.0.0-alpha
```

The matching controller image naming pattern is:

```text
ghcr.io/<REPOSITORY_OWNER>/oci-service-operator-<GROUP>:v2.0.0-alpha
```

The workflow's default `subpackages=all` publish set is:

- `apigateway`
- `containerengine`
- `containerinstances`
- `core-network`
- `database`
- `dataflow`
- `functions`
- `identity`
- `mysql`
- `nosql`
- `objectstorage`
- `opensearch`
- `psql`
- `queue`
- `redis`
- `streaming`
- `vault`

Important scope notes:

- Use the package name from `packages/<group>`, not a guessed OCI service name.
- `core-network` is the published networking split carved out of the broader
  `core` service.
- `database`, `identity`, `objectstorage`, `opensearch`, `redis`, and
  `streaming` currently publish focused bundles whose default-active runtime
  scope is narrower than the full OCI service.
- `apigateway` is published because it exists under `packages/apigateway`,
  even though it is not part of the current default-active selection in
  `internal/generator/config/services.yaml`.
- `core` still has local packaging files in the repo, but the workflow excludes
  it from the default `subpackages=all` batch. Do not assume a published
  `oci-service-operator-core-bundle:v2.0.0-alpha` image unless it was released
  separately.

Each published package uses its own default namespace from
`packages/<group>/metadata.env`, for example
`oci-service-operator-mysql-system` or
`oci-service-operator-core-network-system`.

### Deploy OSOK

Install the OSOK operator in the Kubernetes cluster by selecting a published
package and running:

```bash
$ operator-sdk run bundle ghcr.io/<REPOSITORY_OWNER>/oci-service-operator-<GROUP>-bundle:v2.0.0-alpha
```

Examples:

```bash
$ operator-sdk run bundle ghcr.io/<REPOSITORY_OWNER>/oci-service-operator-mysql-bundle:v2.0.0-alpha
$ operator-sdk run bundle ghcr.io/<REPOSITORY_OWNER>/oci-service-operator-database-bundle:v2.0.0-alpha
$ operator-sdk run bundle ghcr.io/<REPOSITORY_OWNER>/oci-service-operator-core-network-bundle:v2.0.0-alpha
```

If you need the legacy monolithic installation path, the local `Makefile` still
provides `make install-monolith-olm`, but the published examples in this guide
follow the current per-package GHCR bundles.

Upgrade the OSOK operator in the Kubernetes cluster using:

```bash
$ operator-sdk run bundle-upgrade ghcr.io/<REPOSITORY_OWNER>/oci-service-operator-<GROUP>-bundle:v2.0.0-alpha
```

On success, OLM reports installation of the package-specific CSV, for example:

```bash
INFO[0040] OLM has successfully installed "oci-service-operator-mysql.v2.0.0-alpha"
```

### Controller Manager Config

The default kustomize deployment under `config/default` loads controller-runtime
options from `config/manager/controller_manager_config.yaml`. The manager
package builds the `manager-config` ConfigMap from that file and
`config/default/manager_config_patch.yaml` mounts it into the pod while adding
`--config=controller_manager_config.yaml` to the manager container arguments.

When `--config` is present, OSOK treats the file as authoritative instead of
merging it with the default `--metrics-bind-address`,
`--health-probe-bind-address`, or `--leader-elect` flag values. Keep the type
metadata exactly as shown below:

```yaml
apiVersion: controller-runtime.sigs.k8s.io/v1alpha1
kind: ControllerManagerConfig
```

OSOK validates this file strictly during startup. Unknown fields or mismatched
type metadata fail manager startup instead of falling back to defaults. If you
remove `--config` from a custom deployment, the manager reverts to the built-in
command-line defaults from `main_manager_config.go`.

### Undeploy OSOK

The OCI Service Operator for Kubernetes can be undeployed easily using OLM.

```bash
$ operator-sdk cleanup oci-service-operator-<GROUP>
```

Example:

```bash
$ operator-sdk cleanup oci-service-operator-mysql
```

If you installed the legacy monolithic bundle instead of a published
per-package bundle, use:

```bash
$ operator-sdk cleanup oci-service-operator
```

### Customize CA trust bundle

The OCI Service Operator for Kubernetes by default mounts the `/etc/pki` host path so that the host
certificate chains can be used for TLS verification. The default container image is built on top of
Oracle Linux 9 which has the default CA trust bundle under `/etc/pki`. A new container image can be
created with a custom CA trust bundle.
