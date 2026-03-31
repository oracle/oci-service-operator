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

The Stream controller publishes OSOK lifecycle bookkeeping under `status.status.*` and the live OCI Stream read model as top-level fields on `status`.

| Parameter | Description | Type | Mandatory |
| --------- | ----------- | ---- | --------- |
| `status.status.conditions.type` | Lifecycle state of the Stream resource. The following values are valid: <ul><li>**Provisioning** - indicates a Stream is provisioning.</li><li>**Active** - indicates a Stream is active.</li><li>**Failed** - indicates a Stream failed provisioning.</li><li>**Terminating** - indicates a Stream is deleting.</li></ul> | string | no |
| `status.status.conditions.status` | Status of the Stream custom resource during the condition update. | string | no |
| `status.status.conditions.lastTransitionTime` | Last time the Stream condition changed. | string | no |
| `status.status.conditions.message` | Message associated with the current Stream condition. | string | no |
| `status.status.conditions.reason` | Reason associated with the current Stream condition. | string | no |
| `status.status.ocid` | The Stream [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) tracked by OSOK. | string | yes |
| `status.status.message` | Overall OSOK status message for the Stream custom resource. | string | no |
| `status.status.reason` | Overall OSOK status reason for the Stream custom resource. | string | no |
| `status.status.createdAt` | Created time recorded for the Stream custom resource. | string | no |
| `status.status.updatedAt` | Last updated time recorded for the Stream custom resource. | string | no |
| `status.status.requestedAt` | Requested time of the Stream custom resource. | string | no |
| `status.status.deletedAt` | Deleted time of the Stream custom resource. | string | no |
| `status.name` | The name of the stream. Avoid entering confidential information. | string | no |
| `status.id` | The Stream [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) returned by the live OCI read model. | string | yes |
| `status.partitions` | The number of partitions in the stream. | integer | no |
| `status.retentionInHours` | The retention period of the stream, in hours. This field is read-only. | integer | no |
| `status.compartmentId` | The compartment [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the stream. | string | no |
| `status.streamPoolId` | The stream pool [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) that contains the stream. | string | no |
| `status.lifecycleState` | The current OCI lifecycle state of the stream. | string | no |
| `status.timeCreated` | The creation time of the stream in RFC 3339 timestamp format. | string | no |
| `status.messagesEndpoint` | The endpoint used by `StreamClient` to consume or publish messages in the stream. OSOK also materializes this value into the companion endpoint secret after the stream becomes active. | string | no |
| `status.lifecycleStateDetails` | Additional details about the current OCI lifecycle state of the stream. | string | no |
| `status.freeformTags` | Free-form tags returned by OCI for the stream. | object | no |
| `status.definedTags` | Defined tags returned by OCI for the stream. | object | no |

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
NAME                       NAME          STATUS   OCID                        AGE
stream-sample              StreamTest    Active   ocid1.stream.oc1..<id>      4d
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
