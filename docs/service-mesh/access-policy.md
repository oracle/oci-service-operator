- [Introduction](#introduction)

- [Access Policy Specification Parameters](#access-policy-specification-parameters)

- [Access Policy Status Parameters](#access-policy-status-parameters)

- [Create Resource](#create-resource)

- [Get Resource](#get-resource)

- [Get Resource Detailed](#get-resource-detailed)

- [Describe Resource](#describe-resource)

- [Update Resource](#update-resource)

- [Delete Resource](#delete-resource)


## Introduction
Access Policy enables a Mesh Operator to set access rules on Virtual Services in the Service Mesh.
By default we deny-all the requests if there is no access policy.


## Access Policy Specification Parameters

The Complete Specification of the `AccessPolicy` Custom Resource (CR) is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.Name` | The user-friendly name for the AccessPolicy. The name has to be unique within the same mesh. | string | yes       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the AccessPolicy. | string | yes       |
| `spec.description` | Access policies enable administrators to restrict the access of certain services.  | string | no       |
| `spec.mesh.RefOrId` | The ResourceRef(name,namespace of mesh)/Name of the service mesh in which this access policy is created.  | struct | yes       |
| `spec.rules.action`| An enum to specify whether access is `ALLOW` or `DENY` | enum    | yes       |
| `spec.rules.source`| Source resource such as `ALL_VIRTUAL_SERVICES`,`VIRTUAL_SERVICE`, `EXTERNAL_SERVICE` or `INGRESS_GATEWAY`  | struct    | yes       |
| `spec.rules.destination`| Destination resource such as `ALL_VIRTUAL_SERVICES`, `VIRTUAL_SERVICE`, `EXTERNAL_SERVICE`, `INGRESS_GATEWAY` | struct    | yes       |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | map[string]string  | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). | map[string]map[string]string | no |


## Access Policy Status Parameters

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `status.servicemeshstatus.meshId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of Mesh resources | string | yes       |
| `spec.servicemeshstatus.accessPolicyId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the Access Policy resource | string | yes       |
| `spec.servicemeshstatus.Conditions.ServiceMeshConditionType` | Indicates status of the service mesh resource in the control-plane. Allowed values are [`ServiceMeshActive`, `ServiceMeshDependenciesActive`,`ServiceMeshConfigured`] | enum | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Status` | status of the condition, one of True, False, Unknown. | string | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.ObservedGeneration` | observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if metadata.generation is currently 12, but the status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance. | int | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.LastTransitionTime` | lastTransitionTime is the last time the condition transitioned from one status to another. | struct | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Reason` | reason contains a programmatic identifier indicating the reason for the condition's last transition. | string | yes       |
| `spec.servicemeshstatus.Conditions.ResourceCondition.Message` | message is a human readable message indicating details about the transition. | string | yes       |
| `spec.servicemeshstatus.RefIdForRules` | Reference for Rules in mesh | []map[string] | yes       |
| `spec.servicemeshstatus.LastUpdatedTime` | Time when resource was last updated in operator | time.Time     | no       |


### Create Resource

```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: AccessPolicy
metadata:
  name: <sample-access-policy>      # Access Policy name
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: sample-ap     # Access Policy name inside the mesh
  description: This Access Policy
  mesh:
    ref:
      name: <sample-mesh>
  rules:
    - action: <action_type> [ALLOW,DENY]
      source:
        virtualService:
          ref:
            name: <vs-sample-page>
      destination:
          allVirtualServices: {}
    - action: <action_type> [ALLOW,DENY]
      source:
        ingressGateway:
          ref:
            name: <sample-ingress-gateway>
      destination:
        virtualService:
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
kubectl apply -f access-policy.yaml
```


### Get Resource
Once the CR is created, OSOK will reconcile and create Access Policy resource. OSOK will ensure the Custom Resource is available.

The CR can be seen in the cluster as below:
```sh
$ kubectl get accesspolicies -n <namespace>
NAME               ACTIVE   AGE
access-policy1      True     8d
```
### Get Resource Detailed
The CR can be seen in the cluster with detailed information as below:
```sh
$ kubectl get accesspolicies -o wide -n <namespace>
NAME               ACTIVE   CONFIGURED   DEPENDENCIESACTIVE   OCID                                                                                      AGE
access-policy1   True     True         True                 ocid1.accesspolicies.oc1.iad.amaaaaaazueyztqasspcu6d4kh3fvcdsj6lzzawbl63a3ytus3ogzxwaejta   8d
```
### Describe Resource
The CR can be described as below:
```sh
$ kubectl describe accesspolicy <name> -n <namespace>
```

### Update Resource
Update the Custom Resource:
Access Policy CR can be updated as follows:
Change the configuration file as needed.
Save the file.
Run the apply command again.


```yaml
apiVersion: servicemesh.oci.oracle.com/v1beta1
kind: AccessPolicy
metadata:
  name: <sample-access-policy>      # Access Policy name
  namespace: <sample-namespace>
spec:
  compartmentId: ocid1.compartment.oc1..aaa...
  name: sample-ap     # Access Policy name inside the mesh
  description: This Access Policy
  mesh:
    ref:
      name: <updated-sample-mesh>
  rules:
    - action: <action_type> [ALLOW,DENY]
      source:
        virtualService:
          ref:
            name: <updated-vs-sample-page>
      destination:
        type:  
          allVirtualServices: {}
    - action: <action_type> [ALLOW,DENY]
      source:
        ingressGateway:
          ref:
            name: <updated-ingress-gateway>
      destination:
        virtualService:
          ref:
            name: <updated-vs-sample-page>

```

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```


### Delete Resource
Delete the Custom Resource:
```sh
kubectl delete accesspolicy <name> -n <namespace>
```
