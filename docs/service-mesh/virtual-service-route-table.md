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
A table that contains a list of routing rules which are used managing the ingress traffic to a virtual service. It is used to route requests to specific virtual deployments of a virtual service. 
This allows the Application Developer to split traffic based on protocol parameters like HTTP headers, query parameters or gRPC attributes and specify the configuration for retries, timeouts and fault injection on those routes.


## Virtual Service Route Table Specification Parameters
The Complete Specification of the `VirtualServiceRouteTable` Custom Resource (CR) is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.Name` | The user-friendly name for the VirtualServiceRouteTable. The name has to be unique within the same Mesh. | string | yes       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the VirtualServiceRouteTable. | string | yes       |
| `spec.description` |  This field stores description of a particular VirtualServiceRouteTable  | string | no       |
| `spec.VirtualService.RefOrId` | The ResourceRef(name,namespace of mesh)/Name of virtual service in which this virtual deployment is created.  | struct | yes       |
| `spec.Priority` | The priority of the route table. A lower value means a higher priority. The routes are declared based on the priority. | int | no       |
| `spec.RouteRules.VirtualServiceTrafficRouteRule.HttpVirtualServiceTrafficRouteRule.Destinations.VirtualDeployment.RefOrId` | The ResourceRef(name,namespace of mesh)/Name of VirtualDeployment to which routing is done  | struct | no       |
| `spec.RouteRules.VirtualServiceTrafficRouteRule.HttpVirtualServiceTrafficRouteRule.Destinations.Weight` | Weight of routing destination  | int | no       |
| `spec.RouteRules.VirtualServiceTrafficRouteRule.HttpVirtualServiceTrafficRouteRule.Destinations.Port` | Destination Service Port  | int | no       |
| `spec.RouteRules.VirtualServiceTrafficRouteRule.HttpVirtualServiceTrafficRouteRule.Path` | Route to match | string | no       |
| `spec.RouteRules.VirtualServiceTrafficRouteRule.HttpVirtualServiceTrafficRouteRule.IsGrpc` | If true, the rule will check that the content-type header has a application/grpc  | string | no       |
| `spec.RouteRules.VirtualServiceTrafficRouteRule.HttpVirtualServiceTrafficRouteRule.PathType` | Match type for the route [PREFIX]   | enum | no       |
| `spec.RouteRules.VirtualServiceTrafficRouteRule.TcpVirtualServiceTrafficRouteRule.Destinations.VirtualDeployment.RefOrId` | The ResourceRef(name,namespace of mesh)/Name of VirtualDeployment to which routing is done  | struct | no       |
| `spec.RouteRules.VirtualServiceTrafficRouteRule.TcpVirtualServiceTrafficRouteRule.Destinations.Weight` | Weight of routing destination | int | no       |
| `spec.RouteRules.VirtualServiceTrafficRouteRule.TcpVirtualServiceTrafficRouteRule.Destinations.Port` | Destination Service Port   | int | no       |
| `spec.RouteRules.VirtualServiceTrafficRouteRule.TlsPassthroughVirtualServiceTrafficRouteRule.Destinations.VirtualDeployment.RefOrId` | The ResourceRef(name,namespace of mesh)/Name of VirtualDeployment to which routing is done  | struct | no       |
| `spec.RouteRules.VirtualServiceTrafficRouteRule.TlsPassthroughVirtualServiceTrafficRouteRule.Destinations.Weight` | Weight of routing destination  | int | no       |
| `spec.RouteRules.VirtualServiceTrafficRouteRule.TlsPassthroughVirtualServiceTrafficRouteRule.Destinations.Port` | Destination Service Port   | int | no       |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | map[string]string  | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). | map[string]map[string]string | no |

## Virtual Service Route Table Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.servicemeshstatus.virtualServiceId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Service resource | string | yes       |
| `spec.servicemeshstatus.virtualServiceName` | The name of the Virtual Service resource | string | yes       |
| `spec.servicemeshstatus.virtualServiceRouteTableId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Service Route Table resource | string | yes       |
| `spec.servicemeshstatus.Conditions.ServiceMeshConditionType` | Indicates status of the service mesh resource in the control-plane. Allowed values are [`ServiceMeshActive`, `ServiceMeshDependenciesActive`,`ServiceMeshConfigured`] | enum | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Status` | status of the condition, one of True, False, Unknown. | string | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.ObservedGeneration` | observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if metadata.generation is currently 12, but the status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance. | int | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.LastTransitionTime` | lastTransitionTime is the last time the condition transitioned from one status to another. | struct | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Reason` | reason contains a programmatic identifier indicating the reason for the condition's last transition. | string | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Message` | message is a human readable message indicating details about the transition. | string | yes       |
| `spec.servicemeshstatus.VirtualDeploymentIdForRules` | Reference for rules in virtual deployment | [][]string | yes       |
| `spec.servicemeshstatus.LastUpdatedTime` | Time when resource was last updated in operator | time.Time | no       |


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
