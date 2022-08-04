- [Introduction](#introduction)

- [Mesh Specification Parameters](#mesh-specification-parameters)
  
- [Mesh Status Parameters](#mesh-status-parameters)
  
- [Create Resource](#create-resource)

- [Get Resource](#get-resource)

- [Get Resource Detailed](#get-resource-detailed)

- [Describe Resource](#describe-resource)

- [Update Resource](#update-resource)

- [Delete Resource](#delete-resource)


## Introduction
Mesh is the top level container resource that represents the logical boundary of application traffic between the services and deployments that reside within it. 
A mesh can span across multiple platforms such as OKE, OCI VM/BM, Kubernetes on OCI, On-premise VMs, OCI Container Instances and OCI Serverless Kubernetes. 
It also provides a unit of access control.


## Mesh Specification Parameters
The Complete Specification of the `Mesh` Custom Resource (CR) is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.displayName` | The user-friendly name for the Mesh. The name has to be unique within the same namespace. | string | no       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the Mesh. | string | yes       |
| `spec.description` | The description of the Mesh.  | string | no       |
| `spec.mtls` | The mTLS authentication for all virtual services within the mesh.  | [MeshMutualTransportLayerSecurity](#meshmutualtransportlayersecurity) | yes       |
| `spec.certificateAuthorities`| An array of certificate authority resources to use for creating leaf certificates. | [][CertificateAuthority](#certificateauthority)    | yes       |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | map[string]string  | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). | map[string]map[string]string | no |

### MeshMutualTransportLayerSecurity
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `minimum` | A minimum level of mTLS authentication for all virtual services within the mesh. Accepts `DISABLED`, `PERMISSIVE` and `STRICT`.  | enum | yes       |

### CertificateAuthority
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `id`| The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the certificate authority. | string    | yes       |

## Mesh Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `status.meshId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of Mesh resources | string | yes       |
| `status.meshMtls` | Sets a minimum level of mTLS authentication for all virtual services within the mesh. [`DISABLED`, `PERMISSIVE`, `STRICT`] | enum | yes       |
| `status.conditions` | Indicates the condition of the Service mesh resource | [][ServiceMeshCondition](#servicemeshcondition) | yes       |
| `status.opcRetryToken` | Unique token for the request sent for fetching the mesh status | string | yes       |
| `status.lastUpdatedTime` | Time when resource was last updated in operator | time.Time | no       |

### ServiceMeshCondition
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `type` | Indicates status of the service mesh resource in the control-plane. Allowed values are [`ServiceMeshActive`, `ServiceMeshDependenciesActive`,`ServiceMeshConfigured`] | enum | yes       |
| `status` | status of the condition, one of True, False, Unknown. | string | yes       |
| `observedGeneration` | observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if metadata.generation is currently 12, but the status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance. | int | yes       |
| `lastTransitionTime` | lastTransitionTime is the last time the condition transitioned from one status to another. | struct | yes       |
| `reason` | reason contains a programmatic identifier indicating the reason for the condition's last transition. | string | yes       |
| `message` | message is a human readable message indicating details about the transition. | string | yes       |

### Create Resource

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: Mesh
metadata:
  name: <sample-mesh>
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  displayName: <sample-mesh>
  description: Description of your mesh here.
  mtls:
    minimum: <mtls-mode>  [DISABLED. STRICT, PERMISSIVE]
  certificateAuthorities:
    - id: ocid1.certificateauthority.oc1.iad.aaa...
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAGNAMESPACE1>:
      <KEY1>: <VALUE1>
```

Run the following command to create a CR in the cluster:
```sh
kubectl apply -f mesh.yaml 
```

### Get Resource
Once the CR is created, OSOK will reconcile and create Mesh resource. OSOK will ensure the Mesh Custom Resource is available.

The Mesh CR can be seen in the cluster as below:
```sh
$ kubectl get Meshes -n <namespace>
NAME               ACTIVE   AGE
long-canary-mesh   True     8d
```
### Get Resource Detailed
The Mesh CR can be seen in the cluster with detailed information as below:
```sh
$ kubectl get Meshes -o wide -n <namespace>
NAME               ACTIVE   CONFIGURED   DEPENDENCIESACTIVE   OCID                                                                              AGE
long-canary-mesh   True     True         True                 ocid1.mesh.oc1.iad.amaaaaaazueyztqasspcu6d4kh3fvcdsj6lzzawbl63a3ytus3ogzxwaejta   8d
```
### Describe Resource
The Mesh CR can be described as below:
```sh
$ kubectl describe Meshes <NAME_OF_CR_OBJECT> -n <namespace>
```
### Update Resource
Update a Mesh:
Mesh CR can be updated as follows:
Change the configuration file as needed.
Save the file.
Run the apply command again.

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: Mesh
metadata:
  name: <sample-mesh>
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  displayName: <updated-mesh>
  description: <updated description>
  mtls:
    minimum: <updated-mtls-mode> [DISABLED. STRICT, PERMISSIVE]
  certificateAuthorities:
    - id: <certificate-ocid>
```

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

### Delete Resource
Delete a Mesh:
```sh
kubectl delete mesh <name> -n <namespace>
```
