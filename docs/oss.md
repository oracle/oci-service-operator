# Oracle Streaming Service

- [Introduction](#introduction)
- [Create Policies](#create-policies)
- [Stream Specification Parameters](#stream-specification-parameters)
- [Stream Status Parameters](#stream-status-parameters)
- [Provisioning a Stream](#provisioning-a-stream)
- [Current v2 notes](#current-v2-notes)
- [Access Information](#access-information)
- [Delete a Stream](#delete-a-stream)

## Introduction

The [Oracle Streaming service](https://docs.oracle.com/en-us/iaas/Content/Streaming/Concepts/streamingoverview.htm) provides a fully managed, scalable, and durable solution for ingesting and consuming high-volume data streams in real-time. Oracle Streaming is offered via the OCI Service Operator for Kubernetes (OSOK), making it easy for applications to provision and integrate seamlessly.

## Create Policies

**For Instance Principal**
The OCI Service Operator dynamic group should have the `manage` permission for the `stream-family` and `streampools` resource types.

**Sample Policy:**

```plain
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to manage stream-family in compartment <COMPARTMENT_NAME>
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to manage streampools in compartment <COMPARTMENT_NAME>
```

**For User Principal**
The OCI Service Operator user should have the `manage` permission for the `stream-family` and `streampools` resource types.

**Sample Policy:**

```plain
Allow group <SERVICE_BROKER_GROUP> to manage stream-family in compartment <COMPARTMENT_NAME>
Allow group <SERVICE_BROKER_GROUP> to manage streampools in compartment <COMPARTMENT_NAME>
```

## Stream Specification Parameters

The generated v2 `Stream` CR is defined in `api/streaming/v1beta1/stream_types.go`.
Commonly used spec fields are summarized below:

| Parameter | Description | Type | Mandatory |
| --- | --- | --- | --- |
| `spec.name` | The name of the stream. Avoid entering confidential information. | string | yes |
| `spec.partitions` | The number of partitions in the stream. | integer | yes |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment that contains the stream. Use this instead of `spec.streamPoolId` when you are not targeting a stream pool. | string | no |
| `spec.streamPoolId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the stream pool that contains the stream. Use this instead of `spec.compartmentId` when creating the stream in a pool. | string | no |
| `spec.retentionInHours` | The retention period of the stream, in hours. Accepted values are between 24 and 168 (7 days). If omitted, OCI defaults the retention period to 24 hours. | integer | no |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. `Example: {"Department": "Finance"}` | object | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. `Example: {"Operations": {"CostCenter": "42"}}` | object | no |

Do not set both `spec.compartmentId` and `spec.streamPoolId` on the same `Stream`.

## Stream Status Parameters

The `Stream` CR exposes both OSOK control-plane status and mirrored OCI stream fields:

| Parameter | Description | Type | Mandatory |
| --- | --- | --- | --- |
| `status.status.conditions[].type` | Lifecycle condition recorded by OSOK. Common values include `Provisioning`, `Active`, `Failed`, and `Terminating`. | string | no |
| `status.status.conditions[].status` | Kubernetes condition status for the last condition update. | string | no |
| `status.status.conditions[].lastTransitionTime` | Last transition timestamp for the condition entry. | string | no |
| `status.status.conditions[].message` | Human-readable status detail for the last condition entry. | string | no |
| `status.status.conditions[].reason` | Machine-readable reason for the last condition entry. | string | no |
| `status.status.ocid` | OCI identifier recorded by OSOK for the stream. | string | yes |
| `status.status.message` | Overall status message for the CR. | string | no |
| `status.status.reason` | Overall status reason for the CR. | string | no |
| `status.status.createdAt` | Timestamp when OSOK first recorded the resource. | string | no |
| `status.status.updatedAt` | Timestamp for the last OSOK status update. | string | no |
| `status.status.requestedAt` | Timestamp when the reconcile request was recorded. | string | no |
| `status.status.deletedAt` | Timestamp when deletion was recorded. | string | no |
| `status.id` | OCI OCID mirrored on the `Stream` status. | string | no |
| `status.lifecycleState` | Current OCI lifecycle state for the stream. | string | no |
| `status.messagesEndpoint` | The endpoint applications use to publish to or consume from the stream. | string | no |
| `status.streamPoolId` | OCI stream pool OCID associated with the stream, when applicable. | string | no |
| `status.timeCreated` | OCI creation timestamp for the stream. | string | no |

## Provisioning a Stream

The current v2 API uses `streaming.oracle.com/v1beta1` and the `Stream` kind directly.

```yaml
apiVersion: streaming.oracle.com/v1beta1
kind: Stream
metadata:
  name: <CR_OBJECT_NAME>
spec:
  name: <STREAM_NAME>
  partitions: <PARTITION_COUNT>
  retentionInHours: <RETENTION_HOURS>
  streamPoolId: <STREAM_POOL_OCID>
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAG_NAMESPACE1>:
      <KEY1>: <VALUE1>
```

Use `spec.compartmentId` instead of `spec.streamPoolId` when you want OCI to place the stream directly in a compartment rather than in a stream pool.

Run the following command to create the CR in the cluster:

```sh
kubectl apply -f <CREATE_YAML>.yaml
```

Once the CR is created, OSOK reconciles the `Stream`, records the OCI identifier in status, and publishes the stream endpoint when the stream becomes active.

List and inspect the resource with:

```sh
kubectl get streams
kubectl get streams -o wide
kubectl describe stream <NAME_OF_CR_OBJECT>
```

## Current v2 notes

- The current Stream API group is `streaming.oracle.com/v1beta1`.
- The generated v2 `Stream` contract does not expose the retired bind-by-id field from the legacy API.
- Do not author new manifests with the retired pre-v2 Stream API group.
- Manage the stream through the existing `Stream` CR rather than through a separate bind-by-id manifest.
- Check `api/streaming/v1beta1/stream_types.go` for the current spec surface when authoring manifests.

## Access Information

When a `Stream` becomes active, OSOK creates a same-namespace Kubernetes Secret named after the `Stream` CR. The Secret contains one data key:

- `endpoint`

The same endpoint is also mirrored on the CR at `status.messagesEndpoint`.

## Delete a Stream

Delete the `Stream` CR to delete the OCI stream and clean up the endpoint Secret:

```sh
kubectl delete stream <CR_OBJECT_NAME>
```
