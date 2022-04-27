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

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.Name` | The user-friendly name for the VirtualDeployment. The name has to be unique within the same Mesh. | string | yes       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the VirtualDeployment. | string | yes       |
| `spec.description` |  This field stores description of a particular Virtual Deployment  | string | no       |
| `spec.accessLogging.isEnabled`| This configuration determines if logging is enabled  | boolean    | no       |
| `spec.VirtualService.RefOrId` | The ResourceRef(name,namespace of mesh)/Name of virtual service in which this virtual deployment is created.  | struct | yes       |
| `spec.ServiceDiscovery.ServiceDiscoveryType` | Describes the ServiceDiscoveryType for Virtual Deployment. [`DNS`] | enum | yes       |
| `spec.Listener.VirtualDeploymentListener.Protocol` | Type of protocol used in Virtual Deployment. [HTTP, HTTP2, GRPC, TCP, TLS_PASSTHROUGH] | enum | yes       |
| `spec.Listener.VirtualDeploymentListener.Port` | Port on which Virtual Deployment is running.  | int | yes       |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | map[string]string  | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). | map[string]map[string]string | no |

## Virtual Deployment Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `status.servicemeshstatus.meshId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of Mesh resources | string | yes       |
| `spec.servicemeshstatus.virtualServiceId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Service resource | string | yes       |
| `spec.servicemeshstatus.virtualServiceName` | The name of the Virtual Service resource | string | yes       |
| `spec.servicemeshstatus.virtualDeploymentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Deployment resource | string | yes       |
| `spec.servicemeshstatus.Conditions.ServiceMeshConditionType` | Indicates status of the service mesh resource in the control-plane. Allowed values are [`ServiceMeshActive`, `ServiceMeshDependenciesActive`,`ServiceMeshConfigured`] | enum | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Status` | status of the condition, one of True, False, Unknown. | string | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.ObservedGeneration` | observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if metadata.generation is currently 12, but the status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance. | int | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.LastTransitionTime` | lastTransitionTime is the last time the condition transitioned from one status to another. | struct | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Reason` | reason contains a programmatic identifier indicating the reason for the condition's last transition. | string | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Message` | message is a human readable message indicating details about the transition. | string | yes       |
| `spec.servicemeshstatus.LastUpdatedTime` | Time when resource was last updated in operator | time.Time | no       |


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
