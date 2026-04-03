# Oracle Cloud Infrastructure Queue

- [Introduction](#introduction)
- [Queue Specification Parameters](#queue-specification-parameters)
- [Queue Status Parameters](#queue-status-parameters)
- [Provision](#provisioning-a-queue)
- [Update](#updating-a-queue)
- [Endpoint Secret](#endpoint-secret)
- [Out of Scope](#out-of-scope)

## Introduction

The Queue rollout in OSOK currently promotes the top-level OCI `Queue` resource
under:

- `apiVersion: queue.oracle.com/v1beta1`
- `kind: Queue`

The published Queue reconcile path is repo-authored around the generated
service-manager seam:

- create, update, and delete are work-request-backed
- Queue identity is recovered from work-request resources rather than
  display-name-only list matching
- status projects live Queue fields plus work-request tracking fields
- a same-name endpoint Secret is managed only after the Queue reaches `Active`

## Queue Specification Parameters

| Parameter | Description | Type | Mandatory |
| --- | --- | --- | --- |
| `spec.displayName` | User-friendly Queue display name. | string | yes |
| `spec.compartmentId` | OCI compartment OCID. | string | yes |
| `spec.retentionInSeconds` | Message retention period in seconds. Force-new after create. | int | no |
| `spec.visibilityInSeconds` | Default visibility timeout in seconds. Mutable. | int | no |
| `spec.timeoutInSeconds` | Default polling timeout in seconds. Mutable. | int | no |
| `spec.channelConsumptionLimit` | Per-channel consumption limit percentage. Mutable. | int | no |
| `spec.deadLetterQueueDeliveryCount` | Delivery count before DLQ handling. Mutable. | int | no |
| `spec.customEncryptionKeyId` | Customer-managed encryption key OCID. Mutable; empty string clears the existing key. | string | no |
| `spec.freeformTags` | OCI freeform tags. Mutable. | object | no |
| `spec.definedTags` | OCI defined tags. Mutable. | object | no |

Mutable fields map to OCI `UpdateQueueDetails`.

Force-new fields for this first rollout:

- `spec.compartmentId`
- `spec.retentionInSeconds`

If either of those drifts from the live OCI Queue, OSOK rejects the update
instead of silently recreating the Queue.

## Queue Status Parameters

| Parameter | Description | Type |
| --- | --- | --- |
| `status.status.conditions.type` | OSOK lifecycle condition such as provisioning, active, updating, terminating, or failed. | string |
| `status.status.ocid` | OCI Queue OCID tracked by OSOK. | string |
| `status.id` | Observed Queue OCID. | string |
| `status.compartmentId` | Observed compartment OCID. | string |
| `status.displayName` | Observed display name. | string |
| `status.lifecycleState` | Observed OCI Queue lifecycle state. | string |
| `status.messagesEndpoint` | Observed Queue endpoint used for the endpoint Secret. | string |
| `status.retentionInSeconds` | Observed retention period. | int |
| `status.visibilityInSeconds` | Observed visibility timeout. | int |
| `status.timeoutInSeconds` | Observed polling timeout. | int |
| `status.deadLetterQueueDeliveryCount` | Observed dead-letter delivery count. | int |
| `status.customEncryptionKeyId` | Observed encryption key OCID. | string |
| `status.freeformTags` | Observed freeform tags. | object |
| `status.definedTags` | Observed defined tags. | object |
| `status.systemTags` | Observed OCI system tags. | object |
| `status.channelConsumptionLimit` | Observed per-channel consumption limit percentage. | int |
| `status.createWorkRequestId` | In-flight create work request OCID. | string |
| `status.updateWorkRequestId` | In-flight update work request OCID. | string |
| `status.deleteWorkRequestId` | In-flight delete work request OCID. | string |

The work-request fields are stable across requeues so the handwritten runtime
can resume asynchronous OCI operations instead of restarting them.

## Provisioning a Queue

```yaml
apiVersion: queue.oracle.com/v1beta1
kind: Queue
metadata:
  name: example-queue
spec:
  compartmentId: <COMPARTMENT_OCID>
  displayName: example-queue
  retentionInSeconds: 1200
  visibilityInSeconds: 30
  timeoutInSeconds: 20
  deadLetterQueueDeliveryCount: 5
  freeformTags:
    env: dev
```

Apply the resource:

```sh
kubectl apply -f queue.yaml
```

Inspect status:

```sh
kubectl get queues
kubectl describe queue example-queue
```

During provisioning, OSOK stores the create work-request OCID in
`status.createWorkRequestId` and requeues until the Queue work request resolves
to a real Queue OCID and the observed Queue reaches `ACTIVE`.

## Updating a Queue

Update the same `Queue` object by modifying supported mutable fields only.

```yaml
apiVersion: queue.oracle.com/v1beta1
kind: Queue
metadata:
  name: example-queue
spec:
  compartmentId: <COMPARTMENT_OCID>
  displayName: example-queue-updated
  visibilityInSeconds: 45
  timeoutInSeconds: 25
  channelConsumptionLimit: 80
  deadLetterQueueDeliveryCount: 10
  customEncryptionKeyId: ""
```

Notes:

- `customEncryptionKeyId: ""` explicitly clears a previously configured key.
- `compartmentId` and `retentionInSeconds` are not mutable in this rollout and
  are rejected as create-only drift if they differ from the live OCI Queue.
- OSOK persists `status.updateWorkRequestId` while OCI update work is still in
  flight.

## Endpoint Secret

Once the Queue reaches `Active` and `status.messagesEndpoint` is available,
OSOK manages a same-name Secret in the Queue namespace with:

- key: `endpoint`
- value: the observed Queue `messagesEndpoint`

Ownership and mutation rules mirror the promoted Stream behavior:

- OSOK labels owned Secrets with `queue.oracle.com/queue-uid=<queue UID>`
- OSOK uses guarded updates and guarded deletes for owned Secrets
- OSOK adopts only an unlabeled same-name Secret whose data already matches the
  desired endpoint payload
- OSOK skips missing or unowned Secrets during delete cleanup

## Out of Scope

This first Queue rollout is limited to `Queue`.

The following Queue-family resources remain out of scope:

- `Channel`
- `Message`
- `Stats`
- `WorkRequest`
- `WorkRequestError`
- `WorkRequestLog`
