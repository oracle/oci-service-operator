- [Introduction](#introduction)

- [Ingress Gateway Specification Parameters](#ingress-gateway-specification-parameters)

- [Ingress Gateway Status Parameters](#ingress-gateway-status-parameters)

- [Create Resource](#create-resource)

- [Get Resource](#get-resource)

- [Get Resource Detailed](#get-resource-detailed)

- [Describe Resource](#describe-resource)

- [Update Resource](#update-resource)

- [Delete Resource](#delete-resource)



## Introduction
An ingress gateway enables mesh functionality (observability, traffic shaping, security) for user-facing services, 
by applying the defined policies to external traffic entering the mesh.


## Ingress Gateway Specification Parameters

The Complete Specification of the `IngressGateway` Custom Resource (CR) is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.Name` | The user-friendly name for the IngressGateway. The name has to be unique within the same Mesh. | string | yes       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the IngressGateway. | string | yes       |
| `spec.description` |  An ingress gateway allows resources that are outside of a mesh to communicate to resources that are inside the mesh. It sits on the edge of a service mesh receiving incoming HTTP/TCP connections to the mesh.  | string | no       |
| `spec.mesh.RefOrId` |The ResourceRef(name,namespace of mesh)/Name of the service mesh in which this Ingress gateway is created.  | struct | yes       |
| `spec.accessLogging.isEnabled`| This configuration determines if logging is enabled  | boolean    | no       |
| `spec.hosts.name`| A user-friendly name for the host. This name can be used in the ingress gateway route table resource to attach a route to this host. | string    | yes       |
| `spec.hosts.hostnames`| Hostnames of the host. Wildcard hostnames are supported in the prefix form. | []string    | yes       |
| `spec.hosts.listeners.protocol`| Type of protocol used in resource. HTTP, TCP, TLS_PASSTHROUGH | enum    | no       |
| `spec.hosts.listeners.port`| Port in which resource is running | number    | no       |
| `spec.hosts.listeners.mode`| DISABLED, PERMISSIVE, TLS, MUTUAL_TLS  | enum    | no       |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | map[string]string  | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). | map[string]map[string]string | no |

## Ingress Gateway Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `status.servicemeshstatus.meshId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of Mesh resources | string | yes       |
| `spec.servicemeshstatus.ingressGatewayId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Ingress Gateway resource | string | yes       |
| `spec.servicemeshstatus.IngressGatewayMtls` | The OCID of the certificate resource that will be used for mTLS authentication with other virtual services in the mesh. [`DISABLED`, `PERMISSIVE`, `STRICT`] | enum | yes       |
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
kind: IngressGateway
metadata:
  name: <sample-ingress-gateway>    # Name of Ingress Gateway
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: <sample-ig>     # Ingress Gateway name inside the mesh
  description: This Ingress Gateway
  mesh:
    ref:
      name: <sample-mesh>
  hosts:
    - name: samplehost
      hostnames:
        - samplehost.example.com
        - www.samplehost.example.com
      listeners:
        - port: 8080
          protocol: <protocol-type> [HTTP, TCP, TLS_PASSTHROUGH]
          tls:
            mode: <mtls-mode> [DISABLED, PERMISSIVE, TLS, MUTUAL_TLS]
            serverCertificate:
              ociTlsCertificate:
                certificateId: ocid1.certificate.oc1..aaa...
            clientValidation:
              trustedCaBundle:
                ociCaBundle:
                  caBundleId: ocid1.caBundle.oc1..aaa...
              subjectAlternateNames:
                - authorized1.client
                - authorized2.client
        - port: 9090
          protocol: HTTP
          tls:
            mode: TLS
            serverCertificate:
              kubeSecretTlsCertificate:
                secretName: sampleCertSecretName
  accessLogging:
    isEnabled: true
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAGNAMESPACE1>:
      <KEY1>: <VALUE1>
```

Run the following command to create a CR in the cluster:
```sh
kubectl apply -f ingress-gateway.yaml
```

### Get Resource
Once the CR is created, OSOK will reconcile and create Ingress Gateway resource. OSOK will ensure the Custom Resource is available.

The CR can be seen in the cluster as below:
```sh
$ kubectl get ingressgateways -n <namespace>
NAME               ACTIVE   AGE
ingress-gateway1      True     8d
```

### Get Resource Detailed
The CR can be seen in the cluster with detailed information as below:
```sh
$ kubectl get ingressgateways -o wide -n <namespace>
NAME               ACTIVE   CONFIGURED   DEPENDENCIESACTIVE   OCID                                                                                        AGE
ingress-gateway1   True     True         True                 ocid1.ingressgateway.oc1.iad.amaaaaaazueyztqasspcu6d4kh3fvcdsj6lzzawbl63a3ytus3ogzxwaejta   8d
```

### Describe Resource
The CR can be described as below:
```sh
$ kubectl describe ingressgateway <name> -n <namespace>
```

### Update Resource
Update the Custom Resource:
Ingress Gateway CR can be updated as follows:
Change the configuration file as needed.
Save the file.
Run the apply command again.

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: IngressGateway
metadata:
  name: <sample-ingress-gateway>    # Name of Ingress Gateway
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: <updated-ig>     # Ingress Gateway name inside the mesh
  description: This Ingress Gateway
  mesh:
    ref:
      name: <updated-mesh>
  hosts:
    - name: samplehost
      hostnames:
        - <updated-host1>
        - <updated-host2>
      listeners:
        - port: <updated-port>
          protocol: <protocol-type> [HTTP, TCP, TLS_PASSTHROUGH]
          tls:
            mode: <updated-tls-mode> [DISABLED, PERMISSIVE, TLS, MUTUAL_TLS]
            serverCertificate:
              ociTlsCertificate:
                certificateId: ocid1.certificate.oc1..aaa...
            clientValidation:
              trustedCaBundle:
                ociCaBundle:
                  caBundleId: ocid1.caBundle.oc1..aaa...
              subjectAlternateNames:
                - authorized1.client
                - authorized2.client
        - port: 9090
          protocol: HTTP
          tls:
            mode: <updated-tls-mode> [DISABLED, PERMISSIVE, TLS, MUTUAL_TLS]
            serverCertificate:
              kubeSecretTlsCertificate:
                secretName: sampleCertSecretName
  accessLogging:
    isEnabled: <updated-value> [true/false]

```

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

### Delete Resource
Delete the Custom Resource:
```sh
kubectl delete ingressgateway <name> -n <namespace>
```
