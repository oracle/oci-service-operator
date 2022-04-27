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
| `spec.IngressGateway.RefOrId` | The OCID/name of Ingress Gateway. Either name or Id is required | string | yes       |
| `spec.IngressDeployment.Autoscaling.MinPods` | Minimum number of pods in Ingress Deployment | int | yes       |
| `spec.IngressDeployment.Autoscaling.MaxPods` | Maximum number of pods in Ingress Deployment | int | yes       |
| `spec.GatewayListener.Protocol` | Name of protocol | string | yes       |
| `spec.GatewayListener.Port` | Port of Gateway | int | yes       |
| `spec.GatewayListener.ServicePort` | ServicePort in Ingress Deployment | int | no       |
| `spec.GatewayListener.Name` |  An Optional name in Ingress Gateway Deployment for the Listener  | string | no       |
| `spec.IngressGatewayService.Name` | An enum specifying the type of Service ['NodePort', 'LoadBalancer', 'ClusterIP'] | enum | no       |
| `spec.IngressGatewayService.Labels` | Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. |  map[string]string  | no       |
| `spec.IngressGatewayService.Annotations` | an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata.  | map[string]string | no       |
| `spec.Secrets.SecretName` |  Reference to kubernetes secret containing tls certificates/trust-chains for ingress gateway.   | string | no       |

## Ingress Gateway Deployment Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `status.servicemeshstatus.meshId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of Mesh resources | string | yes       |
| `spec.servicemeshstatus.ingressGatewayId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Ingress Gateway resource | string | yes       |
| `spec.servicemeshstatus.ingressGatewayName` | The name of the Ingress Gateway resource | string | yes       |
| `spec.servicemeshstatus.Conditions.ServiceMeshConditionType` | Indicates status of the service mesh resource in the control-plane. Allowed values are [`ServiceMeshActive`, `ServiceMeshDependenciesActive`,`ServiceMeshConfigured`] | enum | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Status` | status of the condition, one of True, False, Unknown. | string | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.ObservedGeneration` | observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if metadata.generation is currently 12, but the status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance. | int | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.LastTransitionTime` | lastTransitionTime is the last time the condition transitioned from one status to another. | struct | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Reason` | reason contains a programmatic identifier indicating the reason for the condition's last transition. | string | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Message` | message is a human readable message indicating details about the transition. | string | yes       |
| `spec.servicemeshstatus.LastUpdatedTime` | Time when resource was last updated in operator | time.Time | no       |


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