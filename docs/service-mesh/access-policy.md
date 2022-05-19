- [Introduction](#introduction)

- [Access Policy Specification Parameters](#access-policy-specification-parameters)

- [Access Policy Status Parameters](#access-policy-status-parameters)

- [Create Resource](#create-resource)

- [Get Resource](#get-resource)

- [Get Resource Detailed](#get-resource-detailed)

- [Describe Resource](#describe-resource)

- [Update Resource](#update-resource)

- [Delete Resource](#delete-resource)


## Introduction
Access Policy enables a Mesh Operator to set access rules on Virtual Services in the Service Mesh.
By default we deny-all the requests if there is no access policy.


## Access Policy Specification Parameters

The Complete Specification of the `AccessPolicy` Custom Resource (CR) is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.name` | The user-friendly name for the AccessPolicy. The name has to be unique within the same mesh. | string | no       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the AccessPolicy. | string | yes       |
| `spec.description` | The description of the AccessPolicy.  | string | no       |
| `spec.mesh` | The service mesh in which this access policy is created. Either `id` or `ref` should be provided. | [RefOrId](#reforid) | yes       |
| `spec.rules`| A list of applicable rules. | [][AccessPolicyRule](#accesspolicyrule)    | no       |
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

### AccessPolicyRule
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `action` | The namespace the service mesh in which this access policy is created. By default, if namespace is not provided, it will use the current resource namespace.  | string | yes       |
| `source` | The source of the traffic this access policy applies to, one of `allVirtualServices`, `virtualService` or `ingressGateway` should be provided. | [TrafficTarget](#traffictarget) | yes       |
| `destination` | The destination for the traffic this access policy applies to, one of `allVirtualServices`, `virtualService` or `externalService` should be provided. | [TrafficTarget](#traffictarget) | yes       |

### TrafficTarget
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `allVirtualServices` | Access policy rule will apply to all virtual services.  | [AllVirtualServices](#allvirtualservices) | no       |
| `virtualService` | Access policy rule will apply to target virtual service. | [RefOrId](#reforid) | no       |
| `ingressGateway` | Access policy rule will apply to target ingress gateway. | [RefOrId](#reforid) | no       |
| `externalService` | Access policy rule will apply to target external service. One of `tcpExternalService`, `httpExternalService` or `httpsExternalService` should be provided. | [ExternalService](#externalservice) |  no       |

### AllVirtualServices
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |

### ExternalService
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `tcpExternalService` | External service with TCP protocol.  | [TcpExternalService](#tcpexternalservice) | no       |
| `httpExternalService` | External service with HTTP protocol. | [HttpExternalService](#httpexternalservice) | no       |
| `httpsExternalService` | External service with HTTPS protocol. | [HttpsExternalService](#httpsexternalservice) | no       |

### TcpExternalService
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `ipAddresses` | IpAddresses of the external service in CIDR notation.  | []string | yes       |
| `ports` | Ports exposed by the external service. If left empty all ports will be allowed. | []int32 | no       |

### HttpExternalService
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `hostnames` | Host names of the external service.  | []string | yes       |
| `ports` | Ports exposed by the external service. If left empty all ports will be allowed. | []int32 | no       |

### HttpsExternalService
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `hostnames` | Host names of the external service.  | []string | yes       |
| `ports` | Ports exposed by the external service. If left empty all ports will be allowed. | []int32 | no       |

## Access Policy Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `status.meshId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of Mesh resources | string | yes       |
| `status.accessPolicyId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Access Policy resource | string | yes       |
| `status.conditions` | Indicates the condition of the Service mesh resource | [][ServiceMeshCondition](#servicemeshcondition) | yes       |
| `status.refIdForRules` | Reference for Rules in mesh | []map[string] | yes       |
| `status.lastUpdatedTime` | Time when resource was last updated in operator | time.Time     | no       |

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
kind: AccessPolicy
metadata:
  name: <sample-access-policy>      # Access Policy name
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: sample-ap     # Access Policy name inside the mesh
  description: This Access Policy
  mesh:
    ref:
      name: <sample-mesh>
  rules:
    - action: ALLOW
      source:
        virtualService:
          ref:
            name: <sample-virtual-service>
      destination:
          allVirtualServices: {}
    - action: ALLOW
      source:
        ingressGateway:
          ref:
            name: <sample-ingress-gateway>
      destination:
        virtualService:
          ref:
            name: <sample-virtual-service>
    - action: ALLOW
      source:
        virtualService:
          ref:
            name: <sample-virtual-service>
      destination:
        externalService:
            httpsExternalService:
                hostnames:
                    - <sample-hostname>
                ports:
                    - <sample-port>
      
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAGNAMESPACE1>:
      <KEY1>: <VALUE1>  
```

Run the following command to create a CR in the cluster:
```sh
kubectl apply -f access-policy.yaml
```


### Get Resource
Once the CR is created, OSOK will reconcile and create Access Policy resource. OSOK will ensure the Custom Resource is available.

The CR can be seen in the cluster as below:
```sh
$ kubectl get accesspolicies -n <namespace>
NAME               ACTIVE   AGE
access-policy1      True     8d
```
### Get Resource Detailed
The CR can be seen in the cluster with detailed information as below:
```sh
$ kubectl get accesspolicies -o wide -n <namespace>
NAME               ACTIVE   CONFIGURED   DEPENDENCIESACTIVE   OCID                                                                                      AGE
access-policy1   True     True         True                 ocid1.accesspolicies.oc1.iad.amaaaaaazueyztqasspcu6d4kh3fvcdsj6lzzawbl63a3ytus3ogzxwaejta   8d
```
### Describe Resource
The CR can be described as below:
```sh
$ kubectl describe accesspolicy <name> -n <namespace>
```

### Update Resource
Update the Custom Resource:
Access Policy CR can be updated as follows:
Change the configuration file as needed.
Save the file.
Run the apply command again.


```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: AccessPolicy
metadata:
  name: <sample-access-policy>      # Access Policy name
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: sample-ap     # Access Policy name inside the mesh
  description: This Access Policy
  mesh:
    ref:
      name: <updated-sample-mesh>
  rules:
    - action: ALLOW
      source:
        virtualService:
          ref:
            name: <updated-virtual-service>
      destination:
        type:  
          allVirtualServices: {}
    - action: ALLOW
      source:
        ingressGateway:
          ref:
            name: <updated-ingress-gateway>
      destination:
        virtualService:
          ref:
            name: <updated-virtual-service>
    - action: ALLOW
      source:
        virtualService:
          ref:
            name: <updated-virtual-service>
      destination:
        externalService:
            httpsExternalService:
                hostnames:
                    - <updated-hostname>
                ports:
                    - <updated-port>

```

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```


### Delete Resource
Delete the Custom Resource:
```sh
kubectl delete accesspolicy <name> -n <namespace>
```
