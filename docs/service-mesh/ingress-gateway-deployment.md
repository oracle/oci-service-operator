- [Introduction](#introduction)

- [Ingress Gateway Deployment Specification Parameters](#ingress-gateway-deployment-specification-parameters)

- [Ingress Gateway Deployment Status Parameters](#ingress-gateway-deployment-status-parameters)

- [Create Resource](#create-resource)

- [Get Resource](#get-resource)

- [Get Resource Detailed](#get-resource-detailed)

- [Describe Resource](#describe-resource)

- [Update Resource](#update-resource)

- [Delete Resource](#delete-resource)


## Introduction
After creation an ingress gateway, there is a deployment of the proxy software to the cluster configured as an ingress gateway (different from Kubernetes Ingress resources).
For this purpose an Ingress Gateway Deployment is created and this offloads the management of the deployment and pods backing the ingress gateway to the OCI Service Mesh operator.


## Ingress Gateway Deployment Specification Parameters
The Complete Specification of the `IngressGatewayDeployment` Custom Resource (CR) is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.ingressGateway` | The Ingress Gateway for which this Ingress Gateway Deployment is created. Either `spec.ingressGateway.id` or `spec.ingressGateway.ref` should be provided. | [RefOrId](#reforid) | yes       |
| `spec.deployment` | Deployment configuration for this Ingress Gateway Deployment | [Deployment](#deployment) | yes      |
| `spec.port` | Port configuration of Ingress Gateway Deployment | [][Port](#port) | yes       |
| `spec.secrect` | Reference to kubernetes secret containing tls certificates/trust-chains for ingress gateway | [][Secret](#secret) | no        |
| `spec.service` | Configuration for service in Ingress Gateway Deployment | [Service](#service) | no        |

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

### Deployment
| Parameter                       | Description                                                                             | Type                        | Mandatory |
|---------------------------------|-----------------------------------------------------------------------------------------|-----------------------------|-----------|
| `autoscaling`                   | Contains information about min and max replicas for Ingress Gateway Deployment Resource | [Autoscaling](#autoscaling) | yes       |
| `labels`                        | Additional label information for Ingress Gateway Deployment                             | string                      | no        |
| `mountCertificateChainFromHost` | Indicates whether to mount `/etc/pki` host path to the container                        | bool                        | no        |

### Autoscaling
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `maxPods` | Maximum number of pods available for Ingress Gateway Deployment Resource | integer | yes       |
| `minPods` | Minimum number of pods available for Ingress Gateway Deployment Resource | integer | yes       |
| `resources` | ResourceRequirements describes the compute resource requirements. | [Resources](#resources) | no        |

### Resources
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `limits` | Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/ | map[string]integer | no       |
| `requests` | Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/ | map[string]integer | no        |

### Port
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `name` | An optional name in Ingress Gateway Deployment for the listener. | string | no        |
| `port` | The port of the Ingress Gateway Deployment. | integer | yes       |
| `protocol` | Type of protocol used in resource. Default value is 'TCP'. | string | yes       |
| `serviceport` | The service port in the Ingress Gateway Deployment. | integer | no        |

### Secret
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `secrectName` | name of the secret, this secret should reside in the same namespace as the gateway | string | yes       |

### Service
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `type` | An enum specifying the ingress methods for a service ['NodePort', 'LoadBalancer', 'ClusterIP'] | enum | yes       |
| `annotation` | Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. More info: http://kubernetes.io/docs/user-guide/annotations | map[string]string | no       |
| `labels` | Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels |  map[string]string  | no       |

## Ingress Gateway Deployment Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `status.meshId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of Mesh resources | string | yes       |
| `status.ingressGatewayId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Ingress Gateway resource | string | yes       |
| `status.ingressGatewayName` | The name of the Ingress Gateway resource | string | yes       |
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

- Create your ingress gateway deployment using the ingress gateway name.

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: IngressGatewayDeployment
metadata:
  name: <sample-ingress-gateway>-deployment
  namespace: <sample-namespace>
spec:
  ingressGateway:
    ref:
      name: <sample-ingress-gateway>
  deployment:
    autoscaling:
      minPods: 1
      maxPods: 1
    mountCertificateChainFromHost: true
  ports:
    - protocol: TCP
      port: 8080
      serviceport: 80
  service:
    type: LoadBalancer
  secrets:
    - secretName: secret-tls-secret
```

- Create your ingress gateway deployment using the ingress gateway OCID.

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: IngressGatewayDeployment
metadata:
  name: <sample-ingress-gateway>-deployment
  namespace: <sample-namespace>
spec:
  ingressGateway:
    id: <ocid_of_referenced_ig_created_from_cli>
  deployment:
    autoscaling:
      minPods: 1
      maxPods: 1
    mountCertificateChainFromHost: true
  ports:
    - protocol: TCP
      port: 8080
      serviceport: 80
  service:
    type: LoadBalancer
  secrets:
    - secretName: secret-tls-secret
```

Run the following command to create a CR in the cluster:
```sh
kubectl apply -f ingress-gateway-deployment.yaml
```

### Get Resource
Once the CR is created, OSOK will reconcile and create Ingress Gateway Deployment resource. OSOK will ensure the Custom Resource is available.

The CR can be seen in the cluster as below:
```sh
$ kubectl get ingressgatewaydeployment -n <namespace>
NAME                     ACTIVE   AGE
bookinfo-ig-deployment   True     13h
```
### Get Resource Detailed
The CR can be seen in the cluster with detailed information as below:
```sh
$ kubectl get ingressgatewaydeployment -o wide -n <namespace>
NAME                     ACTIVE   DEPENDENCIESACTIVE   INGRESSGATEWAYOCID                                                                              INGRESSGATEWAYNAME           AGE
bookinfo-ig-deployment   True     True                 ocid1.meshingressgateway.oc1.iad.amaaaaaa3euexniarylru5siihcgdzddmqv72ayr6odellndis6226gb3iva   canary-mesh-dp-bookinfo-ig   13h
```
### Describe Resource
The CR can be described as below:
```sh
$ kubectl describe ingressgatewaydeployment <name> -n <namespace>
```

### Update Resource
To update an ingress gateway deployment with kubectl:
1. Change the configuration text as needed.
2. Run the apply command again.

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: IngressGatewayDeployment
metadata:
  name: <sample-ingress-gateway>-deployment
  namespace: <sample-namespace>
spec:
  ingressGateway:
    ref:
      name: <updated-ingress-gateway>
  deployment:
    autoscaling:
      minPods: <updated-min-pod-count>
      maxPods: <updated-min-pod-count>
    mountCertificateChainFromHost: false
  ports:
    - protocol: <updated-protocol>
      port: <updated-port>
      serviceport: <updated-service-port>
  service:
    type: LoadBalancer
  secrets:
    - secretName: <updated-secret-name>
```

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

### Delete Resource
Delete an Ingress Gateway Deployment
To delete of a specific ingress gateway deployment in your namespace, use the following command:
```sh
kubectl delete ingressgatewaydeployment <name> -n <namespace>
```