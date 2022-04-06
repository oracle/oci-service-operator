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
A table that contains a list of routing rules which are used managing the ingress traffic to an Ingress Gateway.

## Ingress Gateway Route Table Specification Parameters

The Complete Specification of the `IngressGatewayRouteTable` Custom Resource (CR) is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.Name` | The user-friendly name for the IngressGatewayRouteTable. The name does not have to be unique. | string | yes       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the IngressGatewayRouteTable. | string | yes       |
| `spec.description` | An optional field for describing IngressGatewayRouteTable | string | no       |
| `spec.IngressGateway.RefOrId` | The ResourceRef(name,namespace of mesh)/Name  of Ingress Gateway. One of name or Id is required | string | no       |
| `spec.Priority` | The priority of the route table. A lower value means a higher priority. The routes are declared based on the priority. | int | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.HttpIngressGatewayTrafficRouteRule.IngressGatewayHost.Name` | Ingress Host name to which the route rule attaches  | string | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.HttpIngressGatewayTrafficRouteRule.IngressGatewayHost.Port` |  Ingress Host Port Name  | int | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.HttpIngressGatewayTrafficRouteRule.Destinations.VirtualService.RefOrId` | VirtualService Id or name to which routing is done  | struct | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.HttpIngressGatewayTrafficRouteRule.Destinations.Weight` | Weight of routing destination  | int | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.HttpIngressGatewayTrafficRouteRule.Destinations.Port` | Destination Service Port  | int | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.HttpIngressGatewayTrafficRouteRule.Path` | Route rule path  | string | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.HttpIngressGatewayTrafficRouteRule.IsGrpc` | If true, the rule will check that the content-type header has a application/grpc  | boolean | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.HttpIngressGatewayTrafficRouteRule.PathType` | Match type for the route [PREFIX]  | enum | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.HttpIngressGatewayTrafficRouteRule.IsHostRewriteEnabled` | If true, the hostname will be rewritten to the target virtual deployment's DNS hostname.  | string | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.HttpIngressGatewayTrafficRouteRule.IsPathRewriteEnabled` | If true, the matched path prefix will be rewritten to '/' before being directed to the target virtual deployment.  | string | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.TcpIngressGatewayTrafficRouteRule.IngressGatewayHost.Name` | Ingress Host name to which the route rule attaches  | string | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.TcpIngressGatewayTrafficRouteRule.IngressGatewayHost.Port` | Ingress Host Port Name   | int | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.TcpIngressGatewayTrafficRouteRule.Destinations.VirtualService.RefOrId` | Virtual Service ResourceRef(name,namespace of mesh)/Name Id to which routing is done  | struct | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.TcpIngressGatewayTrafficRouteRule.Destinations.Weight` | Weight of routing destination | int | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.TcpIngressGatewayTrafficRouteRule.Destinations.Port` | Destination Service Port   | int | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.TlsPassthroughIngressGatewayTrafficRouteRule.IngressGatewayHost.Name` | Ingress Host name to which the route rule attaches  | string | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.TlsPassthroughIngressGatewayTrafficRouteRule.IngressGatewayHost.Port` |  Ingress Host Port  | int | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.TlsPassthroughIngressGatewayTrafficRouteRule.Destinations.VirtualService.RefOrId` | VirtualService Id or name to which routing is done  | struct | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.TlsPassthroughIngressGatewayTrafficRouteRule.Destinations.Weight` | Weight of routing destination  | int | no       |
| `spec.RouteRules.IngressGatewayTrafficRouteRule.TlsPassthroughIngressGatewayTrafficRouteRule.Destinations.Port` | Destination Service Port   | int | no       |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | map[string]string  | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). | map[string]map[string]string | no |

## Ingress Gateway Route Table Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.servicemeshstatus.ingressGatewayId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Ingress Gateway resource | string | yes       |
| `spec.servicemeshstatus.ingressGatewayName` | The name of the Ingress Gateway resource | string | yes       |
| `spec.servicemeshstatus.ingressGatewayRouteTableId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Ingress Gateway Route Table resource | string | yes       |
| `spec.servicemeshstatus.Conditions.ServiceMeshConditionType` | Indicates status of the service mesh resource in the control-plane. Allowed values are [`ServiceMeshActive`, `ServiceMeshDependenciesActive`,`ServiceMeshConfigured`] | enum | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Status` | status of the condition, one of True, False, Unknown. | string | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.ObservedGeneration` | observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if metadata.generation is currently 12, but the status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance. | int | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.LastTransitionTime` | lastTransitionTime is the last time the condition transitioned from one status to another. | struct | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Reason` | reason contains a programmatic identifier indicating the reason for the condition's last transition. | string | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Message` | message is a human readable message indicating details about the transition. | string | yes       |
| `spec.servicemeshstatus.VirtualServiceIdForRules` | Reference for rules in virtual service | [][]string | yes       |
| `spec.servicemeshstatus.LastUpdatedTime` | Time when resource was last updated in operator | time.Time | no       |


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
