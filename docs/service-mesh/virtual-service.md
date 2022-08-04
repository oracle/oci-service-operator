- [Introduction](#introduction)

- [Virtual Service Specification Parameters](#virtual-service-specification-parameters)

- [Virtual Service Status Parameters](#virtual-service-status-parameters)

- [Create Resource](#create-resource)

- [Get Resource](#get-resource)

- [Get Resource Detailed](#get-resource-detailed)

- [Describe Resource](#describe-resource)

- [Update Resource](#update-resource)

- [Delete Resource](#delete-resource)


## Introduction
VirtualService represents a customer-managed micro-service in the Mesh. It has its own configuration for the service hostname, TLS certificates (client and server), CA bundles and etc. 
Each virtual service contains multiple versions of the service which are represented by a virtual deployment. 
Additionally, the Virtual Service also contains route tables which are used to route ingress traffic to specific versions of the service.


## Virtual Service Specification Parameters
The Complete Specification of the `VirtualService` Custom Resource (CR) is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.name` | The user-friendly name for the Virtual Service. The name has to be unique within the same Mesh.  | string | no       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the Virtual Service. | string | yes       |
| `spec.description` | Description of the Virtual Service  | string | no       |
| `spec.mesh` | The service mesh in which this Virtual Service is created. Either `spec.mesh.id` or `spec.mesh.ref` should be provided. | [RefOrId](#reforid) | yes       |
| `spec.hosts` | The DNS hostnames of the Virtual Service that is used by its callers. Wildcard hostnames are supported in the prefix form.  | []string | yes       |
| `spec.defaultRoutingPolicy` | Routing policy for the Virtual Service. | [DefaultRoutingPolicy](#defaultroutingpolicy) | no       |
| `spec.mtls` | The mTLS authentication mode to use when receiving requests from other Virtual Services or Ingress Gateways within the mesh. | [Mtls](#mtls) | no       |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | map[string]string  | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). | map[string]map[string]string | no |

### RefOrId
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `ref` | The reference of the resource. | [ResourceRef](#resourceref) | no       |
| `id` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the resource. | string | no       |

### ResourceRef
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `namespace` | The namespace of the resource. By default, if namespace is not provided, it will use the current referring resource namespace.  | string | no       |
| `name` | The name of the resource. | string | yes       |

### DefaultRoutingPolicy
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `type` | Type of the Virtual Service routing policy. Either `ROUND ROBIN` or `DENY` should be provided.  | enum | yes       |

### Mtls
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `mode` | `DISABLED`: Connection is not tunneled. `PERMISSIVE`: Connection can be either plaintext or an mTLS tunnel. `STRICT`: Connection is an mTLS tunnel. Clients without a valid certificate will be rejected.' | enum | yes       |

## Virtual Service Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `status.meshId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of Mesh resources | string | yes       |
| `status.virtualServiceId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Service resource | string | yes       |
| `status.VirtualServiceMtls` | sets mTLS settings used when communicating with other virtual services within the mesh. [`DISABLED`, `PERMISSIVE`, `STRICT`] | enum | yes       |
| `status.conditions` | Indicates the condition of the Service mesh resource | [][ServiceMeshCondition](#servicemeshcondition) | yes       |
| `status.LastUpdatedTime` | Time when resource was last updated in operator | time.Time | no       |

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
kind: VirtualService
metadata:
  name: <vs-sample-page>  # Name of virtual service
  namespace: <sample-namespace>
  labels:
    version: v1
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: <vs-sample>   # Virtual service name inside the mesh
  description: <description text here>
  mesh:
    ref:
      name: <sample-mesh>
  defaultRoutingPolicy:
    type: <routing-policy> [`ROUND ROBIN`, `DENY`]
  hosts:
    - <vs-sample-page>.example.com    # Host name matching vs-name not required
  mtls:
    mode: <mtls-mode> [`DISABLED`, `PERMISSIVE`, `STRICT`] 
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAGNAMESPACE1>:
      <KEY1>: <VALUE1>
```

Run the following command to create a CR in the cluster:
```sh
$ kubectl apply -f virtual-service.yaml
```


### Get Resource
Once the CR is created, OSOK will reconcile and create Virtual Service resource. OSOK will ensure the Custom Resource is available.

The CR can be seen in the cluster as below:
```sh
$ kubectl get virtualservices -n <namespace>
NAME                 ACTIVE   AGE
virtual-service1     True     8d
```

### Get Resource Detailed
The CR can be seen in the cluster with detailed information as below:
```sh
$ kubectl get virtualservices -o wide -n <namespace>
NAME               ACTIVE   CONFIGURED   DEPENDENCIESACTIVE   OCID                                                                                        AGE
virtual-service1   True     True         True                 ocid1.virtualservice.oc1.iad.amaaaaaazueyztqasspcu6d4kh3fvcdsj6lzzawbl63a3ytus3ogzxwaejta   8d
```

### Describe Resource
The CR can be described as below:
```sh
$ kubectl describe virtualservice <name> -n <namespace>
```

### Update Resource
Update the Custom Resource:
Virtual Service CR can be updated as follows:
Change the configuration file as needed.
Save the file.
Run the apply command again.

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: VirtualService
metadata:
  name: <vs-sample-page>  # Name of virtual service
  namespace: <sample-namespace>
  labels:
    version: v1
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: <updated-vs-sample>   # Virtual service name inside the mesh
  description: <description text here>
  mesh:
    ref:
      name: <updated-mesh>
  defaultRoutingPolicy:
    type: <updated-routing-policy> [`ROUND ROBIN`, `DENY`]
  hosts:
    - <vs-sample-page>.example.com    # Host name matching vs-name not required
  mtls:
    mode: <updated-mtls-mode> [`DISABLED`, `PERMISSIVE`, `STRICT`] 
```

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

### Delete Resource
Delete the Custom Resource:
```sh
kubectl delete virtualservice <name> -n <namespace>
```