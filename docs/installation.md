# Installation

* [Pre-Requisites](#pre-requisites)
* [Install Operator SDK](#install-operator-sdk)
* [Install Operator Lifecycle Manager (OLM)](#install-olm)
* [Deploy OCI Service Operator for Kuberentes](#deploy-oci-service-operator-for-kubernetes)

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

### Enable Instance Principal

The OCI Service Operator for Kuberentes needs OCI Instance Principal details to provision and manage OCI services/resources in the customer tenancy. This is the recommended approach for running OSOK within OCI.

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

The OSOK will be deployed in `oci-service-operator-system` namespace. For enabling user principals, we need to create the namespace before deployment.

Create a yaml file using below details
```yaml
apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: oci-service-operator-system
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

Run the below command to create Secret by name `ociCredentials`. (Replace values with your user credentials)

```bash
$ kubectl -n oci-service-operator-system create secret generic ocicredentials \
--from-literal=tenancy=<CUSTOMER_TENANCY_OCID> \
--from-literal=user=<USER_OCID> \
--from-literal=fingerprint=<USER_PUBLIC_API_KEY_FINGERPRINT> \
--from-literal=region=<USER_OCI_REGION> \
--from-literal=passphrase=<PASSPHRASE_STRING> \
--from-file=privatekey=<PATH_OF_USER_PRIVATE_API_KEY>
```

The name of the secret will passed in the `osokConfig` config map which will be created as part of the OSOK deployment. By default the name of the user credential secret is `ocicredentials`. Also, the secret should be created in the `oci-service-operator-system` namespace.

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

### Deploy OSOK

The OCI Service Operator for Kubernetes is packaged as Operator Lifecycle Manager (OLM) Bundle for making it easy to install in Kubernetes Clusters. The bundle can be downloaded as docker image using below command.

```bash
$ docker pull iad.ocir.io/oracle/oci-service-operator-bundle:1.1.0
```

The OSOK OLM bundle contains all the required details like CRDs, RBACs, Configmaps, deployment which will install the OSOK in the kubernetes cluster.


Install the OSOK Operator in the Kubernetes Cluster using below command

```bash
$ operator-sdk run bundle iad.ocir.io/oracle/oci-service-operator-bundle:1.1.0
```

Upgrade the OSOK Operator in the Kubernetes Cluster using below command

```bash
$ operator-sdk run bundle-upgrade iad.ocir.io/oracle/oci-service-operator-bundle:1.1.0
```

The successful installation of the OSOK in your cluster will provide the final message as below:
```bash
INFO[0040] OLM has successfully installed "oci-service-operator.v1.1.0"
```

### Undeploy OSOK

The OCI Service Operator for Kubernetes can be undeployed easily using the OLM

```bash
$ operator-sdk cleanup oci-service-operator
```
