- [Introduction](#introduction)

- [Virtual Deployment Binding Specification Parameters](#virtual-deployment-binding-specification-parameters)

- [Virtual Deployment Binding Status Parameters](#virtual-deployment-binding-status-parameters)

- [Create Resource](#create-resource)

- [Get Resource](#get-resource)

- [Get Resource Detailed](#get-resource-detailed)

- [Describe Resource](#describe-resource)

- [Update Resource](#update-resource)

- [Delete Resource](#delete-resource)


## Introduction
A virtual deployment binding associates the pods in your cluster to a virtual deployment in their mesh. 
This binding resource allows enabling automatic sidecar injection, discover pods backing the virtual deployment for service discovery and automatically upgrading pods that are running an older version of the proxy software.


## Virtual Deployment Binding Specification Parameters
The Complete Specification of the `VirtualDeploymentBinding` Custom Resource (CR) is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.VirtualDeployment.RefOrId` | The ResourceRef(name,namespace of mesh)/Name  Reference to virtual deployment id by ocid or name | struct | yes       |
| `spec.Target.Service.Ref.Name` | A reference to service name the binding belongs to. | string | yes       |
| `spec.Target.Service.Ref.Namespace` | A reference to service name the binding belongs to. | string | yes       |
| `spec.Target.Service.matchLabels` | A set of key value pairs used to match the target | map[string]string | no       |
| `spec.ResourceRequirements.Limits.ResourceList` |  Resource CPU and memory requirements map of resource to Quantity. It has information in resource to Quantity format  | map[string][int] | no       |
| `spec.ResourceRequirements.Requests` |  Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. It has information in resource to Quantity format | map[string][int] | no       |


## Virtual Deployment Binding Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `status.servicemeshstatus.meshId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of Mesh resources | string | yes       |
| `spec.servicemeshstatus.virtualServiceId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Service resource | string | yes       |
| `spec.servicemeshstatus.virtualServiceName` | The name of the Virtual Service resource | string | yes       |
| `spec.servicemeshstatus.virtualDeploymentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Deployment resource | string | yes       |
| `spec.servicemeshstatus.virtualDeploymentName` | The name of the Virtual Deployment | string | yes       |
| `spec.servicemeshstatus.Conditions.ServiceMeshConditionType` | Indicates status of the service mesh resource in the control-plane. Allowed values are [`ServiceMeshActive`, `ServiceMeshDependenciesActive`,`ServiceMeshConfigured`] | enum | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Status` | status of the condition, one of True, False, Unknown. | string | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.ObservedGeneration` | observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if metadata.generation is currently 12, but the status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance. | int | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.LastTransitionTime` | lastTransitionTime is the last time the condition transitioned from one status to another. | struct | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Reason` | reason contains a programmatic identifier indicating the reason for the condition's last transition. | string | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Message` | message is a human readable message indicating details about the transition. | string | yes       |
| `spec.servicemeshstatus.LastUpdatedTime` | Time when resource was last updated in operator | time.Time | no       |



### Create Resource
Resource can be created either by referencing virtual deployment name or virtual deployment OCID.

- Create your virtual deployment binding using the virtual deployment name.

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: VirtualDeploymentBinding
metadata:
  name: <binding-name>
  namespace: <binding-namespace>
spec:
  podUpgradeEnabled: true
  virtualDeployment:
    ref:
      name: <vd-name>-version1 # Virtual Deployment Name
  target:
    service:
      ref:
        name: <kubernetes-service-name> # Name of Kubernetes Service
        namespace: <kubernetes-service-namespace> # Name of Kubernetes Service namespace
      matchLabels: # Labels for the Kubernetes service
        version: v1
```

- Create your virtual deployment binding using the virtual deployment OCID.

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: VirtualDeploymentBinding
metadata:
  name: <binding-name>
  namespace: <binding-namespace>
spec:
  podUpgradeEnabled: true
  virtualDeployment:
    id: <ocid-of-referencing-virtual-deployment-created-from-cli>
  target:
    service:
      ref:
        name: <kubernetes-service-name> # Name of Kubernetes Service
        namespace: <kubernetes-service-namespace> # Name of Kubernetes Service namespace
      matchLabels: # Labels for the Kubernetes service
        version: v1
```

Run the following command to create a CR in the cluster:
```sh
kubectl apply -f virtual-deployment-binding.yaml
```

### Get Resource
Once the CR is created, OSOK will reconcile and create Virtual Deployment Binding resource. OSOK will ensure the Custom Resource is available.

The CR can be seen in the cluster as below:
```sh
$ kubectl get virtualdeploymentbindings -n <namespace>
NAME                     ACTIVE   AGE
details-v1-binding       True     17m
```
### Get Resource Detailed
The CR can be seen in the cluster with detailed information as below:
```sh
$ kubectl get virtualdeploymentbindings -o wide -n <namespace>
NAME                     ACTIVE   DEPENDENCIESACTIVE   VIRTUALDEPLOYMENTOCID                                                                              VIRTUALDEPLOYMENTNAME   AGE
details-v1-binding       True     True                 ocid1.meshvirtualdeployment.oc1.iad.amaaaaaazueyztqaeaaej7nkywy4fhzcem5obheerrx2vkwgobtaq3purmpq   details-v1              18m
```
### Describe Resource
The CR can be described as below:
```sh
$ kubectl describe virtualdeploymentbindings <name> -n <namespace>
```

### Update Resource

To update a virtual deployment binding with kubectl:
1. Change the configuration text as needed.
2. Run the apply command again.

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: VirtualDeploymentBinding
metadata:
  name: <binding-name>
  namespace: <binding-namespace>
spec:
  podUpgradeEnabled: <true/false>
  virtualDeployment:
    ref:
      name: <updated-vd-name>-version1 # Updated Virtual Deployment Name
  target:
    service:
      ref:
        name: <kubernetes-service-name> # Name of Kubernetes Service
        namespace: <kubernetes-service-namespace> # Name of Kubernetes Service namespace
      matchLabels: # Labels for the Kubernetes service
        version: v1
```

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

### Delete Resource
Delete a Virtual Deployment Binding
To delete of a specific virtual deployment binding in your namespace, use the following command:
```sh
kubectl delete virtualdeploymentbindings <name> -n <namespace>
```
