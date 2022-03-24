# Oracle Streaming Service

- [Introduction](#introduction)
- [Create Policies](#create-policies)
- [Streams Service Specification Parameters](#streams-service-specification-parameters)
- [Streams Service Status Parameters](#streams-service-status-parameters)
- [Create a Stream](#create-a-stream)
- [Bind](#binding-to-an-existing-stream)
- [Update](#updating-stream)
- [Delete](#delete-stream)

## Introduction

The [Oracle Streaming service](https://docs.oracle.com/en-us/iaas/Content/Streaming/Concepts/streamingoverview.htm) provides a fully managed, scalable, and durable solution for ingesting and consuming high-volume data streams in real-time. The Oracle Streaming Service is offered via the OCI Service Operator for Kubernetes (OSOK), making it easy for applications to provision and integrate seamlessly.

## Create Policies

**For Instance Principle** 
The OCI Service Operator dynamic group should have the `manage` permission for the `stream-family` and `streampools` resource types.

**Sample Policy:**

```plain
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to manage stream-family in compartment <COMPARTMENT_NAME>
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to manage streampools in compartment <COMPARTMENT_NAME>
```

**For User Principle** 
The OCI Service Operator user should have the `manage` permission for resource type `stream-family` and `streampools`.

**Sample Policy:**

```plain
Allow group <SERVICE_BROKER_GROUP> to manage stream-family in compartment <COMPARTMENT_NAME>
Allow group <SERVICE_BROKER_GROUP> to manage streampools in compartment <COMPARTMENT_NAME>
```

## Streams Service Specification Parameters

The Complete Specification of the `Streams` Custom Resource (CR) is as detailed below:

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
| `status.osokstatus.conditions.type`               | Lifecycle state of the Streams Service. The following values are valid: <ul><li>**Provisioning** - indicates a Stream is provisioning. </li><li>**Active** - indicates a Stream is Active. </li><li>**Failed** - indicates a Stream failed provisioning. </li><li>**Terminating** - indicates a Stream is Deleting. </li></ul>|  string  |  no  |
| `status.osokstatus.conditions.status`             | Status of the Stream Custom Resource during the condition update. |  string  |  no  |
| `status.osokstatus.conditions.lastTransitionTime` | Last time the Stream CR was Updated. |  string  |  no  | 
| `status.osokstatus.conditions.message`            | Message of the status condition of the CR. | string | no | 
| `status.osokstatus.conditions.reason`             | Resource if any of status condition of the CR. | string | no |
| `status.osokstatus.ocid`                          | The Stream [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm). |  string  | yes |
| `status.osokstatus.message`                       | Overall status message of the CR.  |  string  | no  |
| `status.osokstatus.reason`                        | Overall status reason of the CR.   | string | no |
| `status.osokstatus.createdAt`                     | Created time of the Streams Service.            | string | no |  
| `status.osokstatus.updatedAt`                     | Updated time of the Streams Service.            | string | no |
| `status.osokstatus.requestedAt`                   | Requested time of the CR.          | string | no |
| `status.osokstatus.deletedAt`                     | Deleted time of the CR.            | string | no | 

## Create a Stream

The OSOK Stream controller provisions a stream when customer provides mandatory fields to the `spec`. The Message endpoint of the stream created will be created as a secret.

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

Once the CR is created, OSOK will reconcile and create a stream. OSOK will ensure the stream instance is available.

The Stream CR can list the streams in the cluster: 
```sh
$ kubectl get streams
NAME                         STATUS         AGE
stream-sample             Active         4d
```

The Stream CR can list the streams in the cluster with detailed information: 
```sh
$ kubectl get streams -o wide
NAME                         DISPLAYNAME   STATUS         OCID                                   AGE
streams-sample             StreamTest    Active         ocid1.streams.oc1........   4d
```

The Stream CR can be described:
```sh
$ kubectl describe stream <NAME_OF_CR_OBJECT>
```

## Binding to an Existing Stream

OSOK allows you to bind to an existing stream instance. In this case, `Id` is the only required field in the CR `spec`. The message endpoint of the stream will be created as a secret.

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: Stream
metadata:
  name: <CR_OBJECT_NAME>
spec:
  Id: <STREAM_OCID>
```

Run the following command to create a CR that binds to an existing stream instance:

```sh
kubectl apply -f <BIND_YAML>.yaml
```

## Updating Stream

You can update `streamPoolId`, `freeFormTags`, and `definedTags` of the stream instance.

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

Run the following command to create a CR that updates an existing stream instance:

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

## Delete Stream

You can delete the stream instance when they delete the CR.

```sh
$ kubectl delete stream <CR_OBJECT_NAME>
```
