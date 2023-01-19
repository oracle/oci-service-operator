- [Introduction](#introduction)

- [Virtual Deployment Specification Parameters](#virtual-deployment-specification-parameters)

- [Virtual Deployment Status Parameters](#virtual-deployment-status-parameters)

- [Create Resource](#create-resource)

- [Get Resource](#get-resource)

- [Get Resource Detailed](#get-resource-detailed)

- [Describe Resource](#describe-resource)

- [Update Resource](#update-resource)

- [Delete Resource](#delete-resource)

## Introduction
A virtual deployment is a version of a virtual service in the mesh. Conceptually, it maps to a group of instances/pods running a specific version of the actual micro-service managed by the customer. 

## Virtual Deployment Specification Parameters
The Complete Specification of the `VirtualDeployment` Custom Resource (CR) is as detailed below:

| Parameter               | Description                                                                                                                                                                                                                                                                | Type                                  | Mandatory |
|-------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------|-----------|
| `spec.name`             | The user-friendly name for the VirtualDeployment. The name has to be unique within the same Mesh                                                                                                                                                                           | string                                | no        |
| `spec.compartmentId`    | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the VirtualDeployment                                                                                                                                             | string                                | yes       |
| `spec.description`      | Description of the Virtual Deployment                                                                                                                                                                                                                                      | string                                | no        |
| `spec.virtualService`   | The virtual service in which this Virtual Deployment is created. Either `spec.virtualService.id` or `spec.virtualService.ref` should be provided                                                                                                                           | [RefOrId](#reforid)                   | yes       |
| `spec.accessLogging`    | Access Logging configuration for the Virtual Deployment                                                                                                                                                                                                                    | [accessLogging](#accesslogging)       | no        |
| `spec.serviceDiscovery` | Service Discovery configuration for the Virtual Deployment                                                                                                                                                                                                                 | [serviceDiscovery](#servicediscovery) | yes       |
| `spec.listener`         | Listeners for the Virtual Deployment                                                                                                                                                                                                                                       | [][listener](#listener)               | yes       |
| `spec.freeformTags`     | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | map[string]string                     | no        |
| `spec.definedTags`      | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm)                                                                        | map[string]map[string]string          | no        |

### RefOrId
| Parameter | Description                                                                                        | Type                        | Mandatory |
|-----------|----------------------------------------------------------------------------------------------------|-----------------------------|-----------|
| `ref`     | The reference of the resource                                                                      | [ResourceRef](#resourceref) | no        |
| `id`      | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the resource | string                      | no        |

### ResourceRef
| Parameter   | Description                                                                                                                   | Type   | Mandatory |
|-------------|-------------------------------------------------------------------------------------------------------------------------------|--------|-----------|
| `namespace` | The namespace of the resource. By default, if namespace is not provided, it will use the current referring resource namespace | string | no        |
| `name`      | The name of the resource.                                                                                                     | string | yes       |

### AccessLogging
| Parameter   | Description                                                             | Type | Mandatory |
|-------------|-------------------------------------------------------------------------|------|-----------|
| `isEnabled` | boolean value to enable or disable access logging on virtual deployment | bool | no        |


### ServiceDiscovery
| Parameter  | Description                                            | Type   | Mandatory |
|------------|--------------------------------------------------------|--------|-----------|
| `type`     | Type of service discovery for virtual deployment [DNS] | enum   | yes       |
| `hostname` | host name for the Virtual Deployment                   | string | yes       |

### Listener
| Parameter  | Description                                                                       | Type | Mandatory |
|------------|-----------------------------------------------------------------------------------|------|-----------|
| `protocol` | communication protocol for the listener [HTTP, HTTP2, GRPC, TCP, TLS_PASSTHROUGH] | enum | yes       |
| `port`     | port value for the listener on Virtual Deployment                                 | int  | yes       |
| `requestTimeoutInMs`     | The maximum duration in milliseconds for the deployed service to respond to an incoming request through the listener. If provided, the timeout value overrides the default timeout of 15 seconds for the HTTP/HTTP2 listeners, and disabled (no timeout) for the GRPC listeners. The value 0 (zero) indicates that the timeout is disabled. The timeout cannot be configured for the TCP and TLS_PASSTHROUGH listeners. For streaming responses from the deployed service, consider either keeping the timeout disabled or set a sufficiently high value.                                 | int64 (long)  | no       |
| `idleTimeoutInMs`     | The maximum duration in milliseconds for which the request's stream may be idle. The value 0 (zero) indicates that the timeout is disabled. Deployment                                 | int64 (long)  | no       |

## Virtual Deployment Status Parameters
| Parameter                    | Description                                                                                                       | Type                                            | Mandatory |
|------------------------------|-------------------------------------------------------------------------------------------------------------------|-------------------------------------------------|-----------|
| `status.meshId`              | [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of Mesh resource                   | string                                          | yes       |
| `status.virtualServiceId`    | [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Service resource    | string                                          | yes       |
| `status.virtualServiceName`  | Name of the Virtual Service resource                                                                              | string                                          | yes       |
| `status.virtualDeploymentId` | [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Deployment resource | string                                          | yes       |
| `status.conditions`          | Indicates the condition of the Virtual Deployment resource                                                        | [][ServiceMeshCondition](#servicemeshcondition) | yes       |
| `status.lastUpdatedTime`     | Time when resource was last updated in operator                                                                   | time.Time                                       | no        |

### ServiceMeshCondition
| Parameter            | Description                                                                                                                                                                                                                                                      | Type   | Mandatory |
|----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------|-----------|
| `type`               | Indicates status of the service mesh resource in the control-plane. Allowed values are [`ServiceMeshActive`, `ServiceMeshDependenciesActive`,`ServiceMeshConfigured`]                                                                                            | enum   | yes       |
| `status`             | Current status of the condition, one of True, False, Unknown.                                                                                                                                                                                                    | string | yes       |
| `observedGeneration` | the last `metadata.generation` that the condition was set upon. For instance, if `metadata.generation` is currently 12 but the status.conditions[x].observedGeneration is 9 then the condition is out of date with respect to the current state of the instance. | int    | yes       |
| `lastTransitionTime` | Time when the condition last transitioned from one status to another.                                                                                                                                                                                            | struct | yes       |
| `reason`             | A programmatic identifier indicating the reason for the condition's last transition.                                                                                                                                                                             | string | yes       |
| `message`            | A human readable message indicating details about the transition.                                                                                                                                                                                                | string | yes       |


### Create Resource

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: VirtualDeployment
metadata:
  name: <vs-sample-page>-version1   # Name of virtual deployment
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: <vs-sample>-v1  # Virtual deployment name inside the mesh
  description: <description-text>
  virtualService:
    ref:
      name: <vs-sample-page> # Name of parent virtual service
  accessLogging:
    isEnabled: true
  serviceDiscovery:
    type: DNS
    hostname: <vs-sample-page>-version1.example.com
  listener:
    - port: 9080
      protocol: HTTP
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAGNAMESPACE1>:
      <KEY1>: <VALUE1>
```

Run the following command to create a CR in the cluster:
```sh
kubectl apply -f virtual-deployment.yaml
```

### Get Resource
Once the CR is created, OSOK will reconcile and create Virtual Deployment resource. OSOK will ensure the Custom Resource is available.

The CR can be seen in the cluster as below:
```sh
$ kubectl get virtualdeployments -n <namespace>
NAME                    ACTIVE   AGE
virtualdeployment1      True      8d
```

### Get Resource Detailed
The CR can be seen in the cluster with detailed information as below:
```sh
$ kubectl get virtualdeployments -o wide -n <namespace>
NAME                ACTIVE    CONFIGURED   DEPENDENCIESACTIVE   OCID                                                                                           AGE
virtualdeployment1   True     True         True                 ocid1.virtualdeployment.oc1.iad.amaaaaaazueyztqasspcu6d4kh3fvcdsj6lzzawbl63a3ytus3ogzxwaejta   8d
```

### Describe Resource
The CR can be described as below:
```sh
$ kubectl describe virtualdeployment <name> -n <namespace>
```

### Update Resource
Update the Custom Resource:
Virtual Deployment CR can be updated as follows:
Change the configuration file as needed.
Save the file.
Run the apply command again.

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: VirtualDeployment
metadata:
  name: <vs-sample-page>-version1   # Name of virtual deployment
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: <updated-vs-sample>-v1  # Virtual deployment name inside the mesh
  description: <updated text description>
  virtualService:
    ref:
      name: <vs-sample-page> # Name of parent virtual service
  accessLogging:
    isEnabled: <updated-value> [true/false]
  serviceDiscovery:
    type: DNS
    hostname: <vs-sample-page>-version1.example.com
  listener:
    - port: <updated-port> 
      protocol: <updated-protocol> [HTTP, TCP, TLS_PASSTHROUGH]

```

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

### Delete Resource
Delete the Custom Resource:
```sh
kubectl delete virtualdeployment <name> -n <namespace>
```
