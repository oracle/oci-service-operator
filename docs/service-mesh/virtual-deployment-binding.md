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

| Parameter                | Description                                                                                                                             | Type                                          | Mandatory |
|--------------------------|-----------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------|-----------|
| `spec.virtualDeployment` | The virtual deployment to bind the resource with. Either `spec.virtualDeployment.id` or `spec.virtualDeployment.ref` should be provided | [RefOrId](#reforid)                           | yes       |
| `spec.target`            | A target kubernetes service to bind the resource with                                                                                   | string                                        | yes       |
| `spec.resources`         | minimum and maximum compute resource requirements for the sidecar container                                                             | [ResourceRequirements](#resourcerequirements) | no        |

### RefOrId
| Parameter | Description                                                                                        | Type                        | Mandatory |
|-----------|----------------------------------------------------------------------------------------------------|-----------------------------|-----------|
| `ref`     | The reference of the resource                                                                      | [ResourceRef](#resourceref) | no        |
| `id`      | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the resource | string                      | no        |

### Target
| Parameter     | Description                                                                                        | Type                        | Mandatory |
|---------------|----------------------------------------------------------------------------------------------------|-----------------------------|-----------|
| `service.ref` | The kubernetes service reference to bind the resource with                                         | [ResourceRef](#resourceref) | yes       |

### ResourceRef
| Parameter   | Description                                                                                                                   | Type   | Mandatory |
|-------------|-------------------------------------------------------------------------------------------------------------------------------|--------|-----------|
| `namespace` | The namespace of the resource. By default, if namespace is not provided, it will use the current referring resource namespace | string | no        |
| `name`      | The name of the resource.                                                                                                     | string | yes       |

### ResourceRequirements
| Parameter  | Description                                                | Type              | Mandatory |
|------------|------------------------------------------------------------|-------------------|-----------|
| `limits`   | describes the maximum amount of compute resources allowed  | map[enum]quantity | no        |
| `requests` | describes the minimum amount of compute resources required | map[enum]quantity | no        |
ResourceRequirements on a virtualDeploymentBinding can be set in the same way as set on a normal pod container: (Requests and limits)[https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#requests-and-limits]

## Virtual Deployment Binding Status Parameters
| Parameter                      | Description                                                                                                       | Type                                            | Mandatory |
|--------------------------------|-------------------------------------------------------------------------------------------------------------------|-------------------------------------------------|-----------|
| `status.meshId`                | [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of Mesh resource                   | string                                          | yes       |
| `status.virtualServiceId`      | [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Service resource    | string                                          | yes       |
| `status.virtualServiceName`    | Name of the Virtual Service resource                                                                              | string                                          | yes       |
| `status.virtualDeploymentId`   | [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Virtual Deployment resource | string                                          | yes       |
| `status.virtualDeploymentName` | Name of the Virtual Deployment resource                                                                           | string                                          | yes       |
| `status.conditions`            | Indicates the condition of the Virtual Deployment resource                                                        | [][ServiceMeshCondition](#servicemeshcondition) | yes       |
| `status.lastUpdatedTime`       | Time when resource was last updated in operator                                                                   | time.Time                                       | no        |

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
      name: <vd-name> # Virtual Deployment Name
  target:
    service:
      ref:
        name: <kubernetes-service-name> # Name of Kubernetes Service
        namespace: <kubernetes-service-namespace> # Name of Kubernetes Service namespace
  resources:
    requests:
      memory: "64Mi"
      cpu: "250m"
    limits:
      memory: "128Mi"
      cpu: "500m"
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
  resources:
    requests:
      memory: "64Mi"
      cpu: "250m"
    limits:
      memory: "128Mi"
      cpu: "500m"
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
      name: <updated-vd-name> # Updated Virtual Deployment Name
  target:
    service:
      ref:
        name: <kubernetes-service-name> # Name of Kubernetes Service
        namespace: <kubernetes-service-namespace> # Name of Kubernetes Service namespace
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
