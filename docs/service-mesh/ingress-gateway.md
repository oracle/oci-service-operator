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
| `spec.name` | The user-friendly name for the IngressGateway. The name has to be unique within the same Mesh. | string | no       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the IngressGateway. | string | yes       |
| `spec.description` |  The description of the IngressGateway  | string | no       |
| `spec.mesh` | The service mesh in which this ingress gateway is created. Either `id` or `ref` should be provided. | [RefOrId](#reforid) | yes       |
| `spec.hosts` | An array of hostnames and their listener configuration that this ingress gateway will bind to | [Hosts](#hosts) | yes       |
| `spec.accessLogging`| AccessLogging information  | [AccessLogging](#accessLogging) | no       |
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

### Hosts
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `name`| A user-friendly name for the host. This name can be used in the ingress gateway route table resource to attach a route to this host. | string    | yes       |
| `hostnames`| Hostnames of the host. Wildcard hostnames are supported in the prefix form. | []string    | yes       |
| `listeners` | Listener configuration for ingress gateway host | [][Listeners](#listeners) | yes       |

### Listeners
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `protocol`| Type of protocol used in resource. HTTP, TCP, TLS_PASSTHROUGH | enum    | yes       |
| `port`| Port in which resource is running | integer    | yes       |
| `tls`| TLS enforcement config for the ingress listener  | [Tls](#tls)    | no        |

### Tls
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `mode` | DISABLED, PERMISSIVE, TLS, MUTUAL_TLS  | enum    | yes       |
| `clientValidation` | IngressHostClientValidationConfig Resource representing the TLS configuration used for validating client certificates. | [ClientValidation](#clientValidation) | no        |
| `serverCertificate` | TlsCertificate Resource representing the location of the TLS certificate | [ServerCertificate](#serverCertificate) | no        |

### ClientValidation
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `subjectAlternateNames` | A list of alternate names to verify the subject identity in the certificate presented by the client. | []string | no        |
| `trustedCaBundle` | CaBundle Resource representing the trusted CA bundle | [TrustedCaBundle](#trustedCaBundle) | no        |

### TrustedCaBundle
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `kubeSecretCaBundle` | KubeSecretCaBundle CA Bundle from kubernetes secrets | [KubeSecretCaBundle](#kubeSecretCaBundle) | no        |
| `ociCaBundle` | OciCaBundle CA Bundle from OCI Certificates service | [OciCaBundle](#ociCaBundle) | no        |

### KubeSecretCaBundle
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `secretName` | The name of the kubernetes secret for CA Bundle resource. | string | yes       |

### OciCaBundle
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `caBundleId` | The OCID of the CA Bundle resource. | string | yes      |

### ServerCertificate
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `kubeSecretTlsCertificate` | KubeSecretTlsCertificate TLS certificate from kubernetes secrets | [KubeSecretTlsCertificate](#kubeSecretTlsCertificate) | no        |
| `ociTlsCertificate` | OciTlsCertificate TLS certificate from OCI Certificates service | [OciTlsCertificate](#ociTlsCertificate) | no        |

### KubeSecretTlsCertificate
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `secretName` | The name of the leaf certificate kubernetes secret. | string | yes       |

### OciTlsCertificate
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `certificateId` | The OCID of the leaf certificate resource. | string | yes       |

### AccessLogging
| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `isEnabled`| This configuration determines if logging is enabled  | boolean    | no       |


## Ingress Gateway Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `status.meshId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of Mesh resources | string | yes       |
| `status.ingressGatewayId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Ingress Gateway resource | string | yes       |
| `status.IngressGatewayMtls` | The OCID of the certificate resource that will be used for mTLS authentication with other virtual services in the mesh. [`DISABLED`, `PERMISSIVE`, `STRICT`] | enum | yes       |
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
