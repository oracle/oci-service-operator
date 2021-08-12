# Oracle Streaming Service

- [Introduction](#introduction)
- [OCI Permission requirement](#oci-permission-requirement)
- [Streams Service Specification Parameters](#streams-service-specification-parameters)
- [Streams Service Status Parameters](#streams-service-status-parameters)
- [Create Stream](#create-stream)
- [Bind](#binding-to-an-existing-stream)
- [Update](#updating-stream)
- [Delete](#delete-stream)

## Introduction

The [OCI Streaming service](https://docs.oracle.com/en-us/iaas/Content/Streaming/Concepts/streamingoverview.htm) provides a fully managed, scalable, and durable solution for ingesting and consuming high-volume data streams in real-time. Streaming service is also offered via OCI Service Operator thereby making it easy for applications to provision and integrate seamlessly with `streams`.

## OCI Permission requirement

**For Instance Principle** 
The Dynamic group for OCI Service Operator should have permission `manage` for resource type `stream-family` and `streampools`.

**Sample Policy:**

```plain
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to manage stream-family in compartment <COMPARTMENT_NAME>
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to manage streampools in compartment <COMPARTMENT_NAME>
```

**For User Principle** 
The OCI OSOK user for OCI Service Operator should have permission `manage` for resource type `stream-family` and `streampools`.

**Sample Policy:**

```plain
Allow group <SERVICE_BROKER_GROUP> to manage stream-family in compartment <COMPARTMENT_NAME>
Allow group <SERVICE_BROKER_GROUP> to manage streampools in compartment <COMPARTMENT_NAME>
```

## Streams Service Specification Parameters

The Complete Specification of the `streams` Custom Resource is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.Id `           | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the stream                                        | string | No       |
| `spec.name`          | The name of the stream. Avoid entering confidential information.                    | string | Yes       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment that contains the stream. | string | Yes       |
| `spec.partitions`   | The number of partitions in the stream.                        | number | Yes       |
| `spec.retentionInHours` | The retention period of the stream, in hours. Accepted values are between 24 and 168 (7 days). If not specified, the stream will have a retention period of 24 hours. | number | No       |
| `spec.streamPoolId`  | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the stream pool that contains the stream.  | string | Yes |
| `spec.freeFormTags`  | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | object | No        |
| `spec.definedTags`   | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Operations": {"CostCenter": "42"}}` | object | No        |

## Streams Service Status Parameters

| Parameter                                         | Description                                                         | Type   | Mandatory |
| --------------------------------------------------| ------------------------------------------------------------------- | ------ | --------- |
| `status.osokstatus.conditions.type`               | Lifecycle state of the Autonomous Database Service. The following values are valid: <ul><li>**Provisioning** - indicates an Autonomous database is provisioning. </li><li>**Active** - indicates an Autonomous Data Service is Active. </li><li>**Failed** - indicates an Autonomous Data Service failed provisioning. </li><li>**Terminating** - indicates an Autonomous Data Service is Deleting. </li></ul>|  string  |  no  |
| `status.osokstatus.conditions.status`             | Status of the Autonomous Database Custom Resource during the condition update. |  string  |  no  |
| `status.osokstatus.conditions.lastTransitionTime` | Last time the Autonomous Database CR was Updated. |  string  |  no  | 
| `status.osokstatus.conditions.message`            | Message of the status condition of the CR. | string | no | 
| `status.osokstatus.conditions.reason`             | Resource if any of status condition of the CR. | string | no |
| `status.osokstatus.ocid`                          | The Autonomous Database [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm). |  string  | yes |
| `status.osokstatus.message`                       | Overall status message of the CR.  |  string  | no  |
| `status.osokstatus.reason`                        | Overall status reason of the CR.   | string | no |
| `status.osokstatus.createdAt`                     | Created time of the Autonomous Database Service.            | string | no |  
| `status.osokstatus.updatedAt`                     | Updated time of the Autonomous Database Service.            | string | no |
| `status.osokstatus.requestedAt`                   | Requested time of the CR.          | string | no |
| `status.osokstatus.deletedAt`                     | Deleted time of the CR.            | string | no | 

## Create Stream

The OSOK Stream controller automatically provisions a Stream when customer provides mandatory fields to the `spec`. The Message endpoint of the stream created will be created as a secret.

Following is a sample CR yaml for Stream.

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: Stream
metadata:
  name: <CR_OBJECT_NAME>
spec:
  compartmentId: <COMPARTMENT_OCID>
  name: <STREAM_NAME>
  partitions: <PARTITION_COUNT>
  retentionInHours: <RETENTION_HOURS>
# Either compartmentId or streamPoolId should be provided.  
  streamPoolId: <STREAM_POOL_OCID>
  freeformTags:
      <KEY1>: <VALUE1>
    definedTags:
      <TAGNAMESPACE1>:
        <KEY1>: <VALUE1>
```

Run the following command to create a CR to the cluster:
```sh
kubectl apply -f <CREATE_YAML>.yaml
```

Once the CR is created, the OSOK will Reconcile and creates a Stream. The OSOK will ensure the Stream instance is Available.

The Stream CR can list the streams in the cluster as below: 
```sh
$ kubectl get streams
NAME                         STATUS         AGE
stream-sample             Active         4d
```

The Stream CR can list the streams in the cluster with detailed information as below: 
```sh
$ kubectl get streams -o wide
NAME                         DISPLAYNAME   STATUS         OCID                                   AGE
streams-sample             StreamTest    Active         ocid1.streams.oc1........   4d
```

The Stream CR can be describe as below:
```sh
$ kubectl describe stream <NAME_OF_CR_OBJECT>
```

## Binding to an Existing Stream

The OSOK allows customers to binds to an existing Stream instance. In this case, `Id` is the only required field in the CR `spec`. The message endpoint of the stream will be created as a secret.

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: Stream
metadata:
  name: <CR_OBJECT_NAME>
spec:
  Id: <STREAM_OCID>
```

Run the following command to create a CR that binds to an existing Stream instance:

```sh
kubectl apply -f <BIND_YAML>.yaml
```

## Updating Stream

Customers can update the Stream instance. Only `streamPoolId`, `freeFormTags` and `definedTags` can be updated in this case.

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: Stream
metadata:
  name: <CR_OBJECT_NAME>
spec:
  Id: <STREAM_OCID>   
  streamPoolId: <STREAM_POOL_OCID>
  freeformTags:
      <KEY1>: <VALUE1>
    definedTags:
      <TAGNAMESPACE1>:
        <KEY1>: <VALUE1>
```

Run the following command to create a CR that updates an existing Stream instance:

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

## Delete Stream

Customers can delete the Stream instance when they delete the CR.

```sh
$ kubectl delete stream <CR_OBJECT_NAME>
```
