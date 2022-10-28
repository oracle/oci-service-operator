- [Introduction](#introduction)

- [Virtual Service Route Table Specification Parameters](#virtual-service-route-table-specification-parameters)

- [Virtual Service Route Table Status Parameters](#virtual-service-route-table-status-parameters)

- [Create Resource](#create-resource)

- [Get Resource](#get-resource)

- [Get Resource Detailed](#get-resource-detailed)

- [Describe Resource](#describe-resource)

- [Update Resource](#update-resource)

- [Delete Resource](#delete-resource)


## Introduction
A table that contains a list of routing rules which are used for managing the ingress traffic to a virtual service. It is used to route requests to specific virtual deployments of a virtual service.


## Virtual Service Route Table Specification Parameters
The Complete Specification of the `VirtualServiceRouteTable` Custom Resource (CR) is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.name` | The user-friendly name for the VirtualServiceRouteTable. The name has to be unique within the same Mesh. | string | no       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the Virtual Service Route Table. | string | yes       |
| `spec.description` | Description of the virtual service route table  | string | no       |
| `spec.virtualService` | The Virtual Service in which this Virtual Service Route Table is created. Either `spec.virtualService.id` or `spec.virtualService.ref` should be provided. | [RefOrId](#reforid) | yes       |
| `spec.priority` | The priority of the Virtual Service Route Table. A lower value means a higher priority. The routes are declared based on the priority. | int | no       |
| `spec.routeRules` | A list of rules for routing incoming Virtual Service traffic to Virtual Deployments. A minimum of 1 rule must be specified. One of `httpRoute`, `tcpRoute` or `tlsPassthroughRoute` should be provided. | [][RouteRule](#routerule) | yes       |
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

### RouteRule
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `httpRoute` | Rule for routing incoming Virtual Service traffic with HTTP protocol. | [HttpRoute](#httproute) | no       |
| `tcpRoute` | Rule for routing incoming Virtual Service traffic with TCP protocol. | [TcpRoute](#tcproute) | no       |
| `tlsPassthroughRoute` | Incoming encrypted traffic is passed "as is" to the application which manages TLS on its own. | [TlsPassthroughRoute](#tlspassthroughroute) | no       |

### HttpRoute
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `destinations` | The destination of the request. At least 1 destination must be specified. | [][Destination](#destination) | yes       |
| `isGrpc` | If true, the rule will check that the content-type header has a application/grpc or one of the various application/grpc+ values. | boolean | no       |
| `path` | Http route to match. | string | no       |
| `pathType` | Match type for the route. Only PREFIX is supported. | enum | no       |

### TcpRoute
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `destinations` | The destination of the request. At least 1 destination must be specified. | [][Destination](#destination) | yes       |

### TlsPassthroughRoute
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `destinations` | The destination of the request. At least 1 destination must be specified. | [][Destination](#destination) | yes       |

### Destination
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `port` | Destination service port. | int | no       |
| `virtualDeployment` | The Virtual Deployment for the target destination. Either `spec.virtualDeployment.id` or `spec.virtualDeployment.ref` should be provided. | [RefOrId](#reforid) | yes       |
| `weight` | Weight of routing destination. Value can be 0 - 100 but sum of weight destinations should equal 100. | int | yes       |

## Virtual Service Route Table Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `status.virtualServiceId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Service resource | string | yes       |
| `status.virtualServiceName` | The name of the Virtual Service resource | string | yes       |
| `status.virtualServiceRouteTableId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Service Route Table resource | string | yes       |
| `status.VirtualDeploymentIdForRules` | Reference for rules in virtual deployment | [][]string | yes       |
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
kind: VirtualServiceRouteTable
metadata:
  name: <vs-sample-page>-rt     # Name of the virtual service route table
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: <vs-sample>-routes   # Virtual service route table name inside the mesh
  description: <description text here>
  virtualService:
    ref:
      name: <vs-sample-page>    # Parent virtual service
  routeRules:
    - httpRoute:
        destinations:
          - virtualDeployment:
              ref:
                name: <vs-sample-page>-version1
            weight: 100
        isGrpc: <boolean-value> [true/false]
        path: /foo,
        pathType: PREFIX
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAGNAMESPACE1>:
      <KEY1>: <VALUE1>
```

Run the following command to create a CR in the cluster:
```sh
kubectl apply -f virtual-service-rt.yaml
```

### Get Resource
Once the CR is created, OSOK will reconcile and create Virtual Service Route Table resource. OSOK will ensure the Custom Resource is available.

The CR can be seen in the cluster as below:
```sh
$ kubectl get virtualserviceroutetables -n <namespace>
NAME                              ACTIVE   AGE
virtual-service-route-table1      True     8d
```

### Get Resource Detailed
The CR can be seen in the cluster with detailed information as below:
```sh
$ kubectl get virtualserviceroutetables -o wide -n <namespace>
NAME                            ACTIVE   CONFIGURED   DEPENDENCIESACTIVE   OCID                                                                                                   AGE
virtual-service-route-table1    True     True         True                 ocid1.virtualserviceroutetable.oc1.iad.amaaaaaazueyztqasspcu6d4kh3fvcdsj6lzzawbl63a3ytus3ogzxwaejta    8d
```

### Describe Resource
The CR can be described as below:
```sh
$ kubectl describe virtualserviceroutetable <name> -n <namespace>
```

### Update Resource
Update the Custom Resource:
Virtual Service Route Table CR can be updated as follows:
Change the configuration file as needed.
Save the file.
Run the apply command again.

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: VirtualServiceRouteTable
metadata:
  name: <vs-sample-page>-rt     # Name of the virtual service route table
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: <updated-vs-sample>-routes   # Virtual service route table name inside the mesh
  description: <updated description text here>
  virtualService:
    ref:
      name: <updated vs-sample-page>    # Parent virtual service
  routeRules:
    - httpRoute:
        destinations:
          - virtualDeployment:
              ref:
                name: <updated- vs-sample-page>-version1
            weight: 100
        isGrpc: <updated-boolean-value> [true/false]
        path: /foo,
        pathType: PREFIX
```

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

### Delete Resource
Delete the Custom Resource:
```sh
kubectl delete virtualserviceroutetable <name> -n <namespace>
```
