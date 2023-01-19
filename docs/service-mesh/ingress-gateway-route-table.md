- [Introduction](#introduction)

- [Ingress Gateway Route Table Specification Parameters](#ingress-gateway-route-table-specification-parameters)

- [Ingress Gateway Route Table Status Parameters](#ingress-gateway-route-table-status-parameters)

- [Create Resource](#create-resource)

- [Get Resource](#get-resource)

- [Get Resource Detailed](#get-resource-detailed)

- [Describe Resource](#describe-resource)

- [Update Resource](#update-resource)

- [Delete Resource](#delete-resource)

## Introduction
A table that contains a list of routing rules which are used for managing the ingress traffic to an Ingress Gateway.

## Ingress Gateway Route Table Specification Parameters

The Complete Specification of the `IngressGatewayRouteTable` Custom Resource (CR) is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.name` | The user-friendly name for the IngressGatewayRouteTable. The name does not have to be unique. | string | no       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the IngressGatewayRouteTable. | string | yes       |
| `spec.description` | An optional field for describing IngressGatewayRouteTable | string | no       |
| `spec.ingressGateway` | The Ingress Gateway for which this Ingress Gateway Route Table is created. Either `spec.ingressGateway.id` or `spec.ingressGateway.ref` should be provided. | [RefOrId](#reforid) | yes       |
| `spec.priority` | The priority of the route table. A lower value means a higher priority. The routes are declared based on the priority. | int | no       |
| `spec.routeRules` | Route rules for the ingress traffic. | [][RouteRules](#routeRules) | yes       |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | map[string]string  | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). | map[string]map[string]string | no |

### RefOrId
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `ref` | The reference of the resource. | [ResourceRef](#resourceref) | no       |
| `id` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the resource. | string | no        |

### ResourceRef
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `namespace` | The namespace of the resource. By default, if namespace is not provided, it will use the current referring resource namespace.  | string | no        |
| `name` | The name of the resource. | string | yes       |

### RouteRules
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `httpRoute` | HttpIngressGatewayTrafficRouteRule Rule for routing incoming ingress gateway traffic with HTTP protocol | [HttpRoute](#httpRoute) | no        |
| `tcpRoute` | TcpIngressGatewayTrafficRouteRule Rule for routing incoming ingress gateway traffic with TCP protocol | [TcpRoute](#tcpRoute) | no        |
| `tlsPassthroughRoute` | TlsPassthroughIngressGatewayTrafficRouteRule Rule for routing incoming ingress gateway traffic with TLS_PASSTHROUGH | [TlsPassthroughRoute](#tlsPassthroughRoute) | no        |

### HttpRoute
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `destinations` | The destination of the request. | [][Destinations](#destinations) | yes       |
| `ingressGatewayHost` | IngressGatewayHostRef The ingress gateway host to which the route rule attaches. If not specified, the route rule gets attached to all hosts on the ingress gateway. | [IngressGatewayHost](#ingressGatewayHost) | no        |
| `path` | Route rule path  | string | no       |
| `isGrpc` | If true, the rule will check that the content-type header has a application/grpc  | boolean | no        |
| `pathType` | Match type for the route [PREFIX]  | enum | no        |
| `isHostRewriteEnabled` | If true, the hostname will be rewritten to the target virtual deployment's DNS hostname.  | string | no        |
| `isPathRewriteEnabled` | If true, the matched path prefix will be rewritten to '/' before being directed to the target virtual deployment.  | string | no        |
| `requestTimeoutInMs` | The maximum duration in milliseconds for the target service to respond to a request. If provided, the timeout value overrides the default timeout of 15 seconds for the HTTP based route rules, and disabled (no timeout) when 'isGrpc' is true. The value 0 (zero) indicates that the timeout is disabled. For streaming responses from the target service, consider either keeping the timeout disabled or set a sufficiently high value.  | int64 (long) | no        |

### TcpRoute
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `destinations` | The destination of the request. | [][Destinations](#destinations) | yes       |
| `ingressGatewayHost` | IngressGatewayHostRef The ingress gateway host to which the route rule attaches. If not specified, the route rule gets attached to all hosts on the ingress gateway. | [IngressGatewayHost](#ingressGatewayHost) | no        |

### TlsPassthroughRoute
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `destinations` | The destination of the request. | [][Destinations](#destinations) | yes       |
| `ingressGatewayHost` | IngressGatewayHostRef The ingress gateway host to which the route rule attaches. If not specified, the route rule gets attached to all hosts on the ingress gateway. | [IngressGatewayHost](#ingressGatewayHost) | no        |

### Destinations
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `port` | Destination service's port | integer | no        |
| `virtualService` | The Virtual Service for which this destination is created. Either `spec.virtualService.id` or `spec.virtualService.ref` should be provided. | [RefOrId](#reforid) | yes      |
| `weight` | Amount of traffic flows to a specific Virtual Service | integer | no        |

### IngressGatewayHost
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `name` | Name of the ingress gateway host | string | yes       |
| `port` | Port of the ingress gateway host to select. Leave empty to select all ports of the host. | integer | no        |


## Ingress Gateway Route Table Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `status.ingressGatewayId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Ingress Gateway resource | string | yes       |
| `status.ingressGatewayName` | The name of the Ingress Gateway resource | string | yes       |
| `status.ingressGatewayRouteTableId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Ingress Gateway Route Table resource | string | yes       |
| `status.conditions` | Indicates the condition of the Service mesh resource | [][ServiceMeshCondition](#servicemeshcondition) | yes       |
| `status.VirtualServiceIdForRules` | Reference for rules in virtual service | [][]string | yes       |
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
kind: IngressGatewayRouteTable
metadata:
  name: <sample-ingress-gateway>-route-table    # Name of Ingress Gateway Route Table
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: <sample-ig-rt>       # Ingress Gateway Route Table name inside the mesh
  description: This Ingress Gateway Route Table
  ingressGateway:
    ref:
      name: <sample-ingress-gateway>
  routeRules:
    - httpRoute:
        ingressGatewayHost:
          name: samplehost
        path: /foo
        pathType: PREFIX
        isGrpc: false
        destinations:
          - virtualService:
              ref:
                name: <vs-sample-page>
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAGNAMESPACE1>:
      <KEY1>: <VALUE1>
```

Run the following command to create a CR in the cluster:
```sh
kubectl apply -f ingress-gateway-rt.yaml
```

### Get Resource
Once the CR is created, OSOK will reconcile and create Ingress Gateway Route Table resource. OSOK will ensure the Custom Resource is available.

The CR can be seen in the cluster as below:
```sh
$ kubectl get ingressgatewayroutetables -n <namespace>
NAME                           ACTIVE   AGE
ingressgatewayroutetable1      True     8d
```

### Get Resource Detailed
The CR can be seen in the cluster with detailed information as below:
```sh
$ kubectl get ingressgatewayroutetables -o wide -n <namespace>
NAME                        ACTIVE   CONFIGURED   DEPENDENCIESACTIVE   OCID                                                                                                   AGE
ingressgatewayroutetable1   True     True         True                 ocid1.ingressgatewayroutetables.oc1.iad.amaaaaaazueyztqasspcu6d4kh3fvcdsj6lzzawbl63a3ytus3ogzxwaejta   8d
```

### Describe Resource
The CR can be described as below:
```sh
$ kubectl describe ingressgatewayroutetable <name> -n <namespace>
```

### Update Resource
Update the Custom Resource:
Ingress Gateway Route Table CR can be updated as follows:
Change the configuration file as needed.
Save the file.
Run the apply command again.

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: IngressGatewayRouteTable
metadata:
  name: <sample-ingress-gateway>-route-table    # Name of Ingress Gateway Route Table
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: <updated-ig-rt>       # Ingress Gateway Route Table name inside the mesh
  description: <updated text here>
  ingressGateway:
    ref:
      name: <updated-ingress-gateway>
  routeRules:
    - httpRoute:
        ingressGatewayHost:
          name: samplehost
        path: <updated-path>
        pathType: PREFIX
        isGrpc: false
        destinations:
          - virtualService:
              ref:
                name: <vs-sample-page>

```

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```


### Delete Resource
Delete the Custom Resource:
```sh
kubectl delete ingressgatewayroutetable <name> -n <namespace>
```
