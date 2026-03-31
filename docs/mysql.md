# Oracle MySQL Database Service

- [Introduction](#introduction)
- [MySQL DbSystem Pre-requisites](#pre-requisites-for-setting-up-mysql-dbsystems)
- [MySQL DbSystem Specification Parameters](#mysql-dbsystem-specification-parameters)
- [MySQL DbSystem Status Parameters](#mysql-dbsystem-status-parameters)
- [Provision](#provisioning-a-mysql-dbsystem)
- [Update](#updating-a-mysql-dbsystem)
- [Kubernetes Secrets](#kubernetes-secrets)

## Introduction

[Oracle MySQL Database Service](https://www.oracle.com/mysql/) is a fully
managed database service that lets developers provision and operate MySQL DB
Systems on OCI.

The generator-owned v2 mysql API now uses:

- `apiVersion: mysql.oracle.com/v1beta1`
- `kind: DbSystem`

The legacy mysql compatibility surface is no longer published.

## Pre-requisites for setting up MySQL DB Systems

If this is your first time using MySQL Database Service, ensure your tenancy
administrator has performed the following tasks.

### Create VCN/Subnets

- Follow the [Virtual Networking Quickstart](https://docs.oracle.com/en-us/iaas/Content/Network/Tasks/quickstartnetworking.htm) to create a VCN and subnets.
- Prefer placing the MySQL DB System in the same VCN as the Kubernetes cluster.
- Review the [MySQL networking setup guide](https://docs.oracle.com/en-us/iaas/mysql-database/doc/networking-setup-mysql-db-systems.html#MYAAS-GUID-2B4F78DD-72D3-45BA-8F6A-AC5E3A11B729).

### Create Policies

Create policies in the root compartment with the following statements.

When using Instance Principals:

```plain
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to {SUBNET_READ, SUBNET_ATTACH, SUBNET_DETACH, VCN_READ, COMPARTMENT_INSPECT} in [ tenancy | compartment <compartment_name> | compartment id <compartment_ocid> ]
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to manage mysql-family in [ tenancy | compartment <compartment_name> | compartment id <compartment_ocid> ]
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to use tag-namespaces in tenancy
```

When using User Principals:

```plain
Allow group <OSOK_GROUP> to {SUBNET_READ, SUBNET_ATTACH, SUBNET_DETACH, VCN_READ, COMPARTMENT_INSPECT} in [ tenancy | compartment <compartment_name> | compartment id <compartment_ocid> ]
Allow group <OSOK_GROUP> to manage mysql-family in [ tenancy | compartment <compartment_name> | compartment id <compartment_ocid> ]
Allow group <OSOK_GROUP> to use tag-namespaces in tenancy
```

Without these policies, the service will not function correctly.

## MySQL DbSystem Specification Parameters

The published CRD is `dbsystems.mysql.oracle.com`, and the top-level kind is
`DbSystem`. The generated v2 surface does not preserve the old `spec.id` bind
alias or the legacy kind name.

| Parameter | Description | Type | Mandatory |
| --- | --- | --- | --- |
| `spec.compartmentId` | OCID of the compartment. | string | yes |
| `spec.shapeName` | Shape name for the DB System. | string | yes |
| `spec.subnetId` | OCID of the subnet associated with the DB System. | string | yes |
| `spec.displayName` | User-friendly display name. | string | no |
| `spec.description` | User-provided description. | string | no |
| `spec.isHighlyAvailable` | Whether the DB System is highly available. | boolean | no |
| `spec.availabilityDomain` | Preferred availability domain. | string | no |
| `spec.faultDomain` | Preferred fault domain. | string | no |
| `spec.configurationId` | OCID of the MySQL configuration to use. | string | no |
| `spec.mysqlVersion` | Specific MySQL version identifier. | string | no |
| `spec.adminUsername` | Administrative username secret reference. Set `spec.adminUsername.secret.secretName` to a Secret in the same namespace that contains a `username` key. | object | no |
| `spec.adminPassword` | Administrative password secret reference. Set `spec.adminPassword.secret.secretName` to a Secret in the same namespace that contains a `password` key. | object | no |
| `spec.dataStorageSizeInGBs` | Initial data volume size in GBs. | int | no |
| `spec.hostnameLabel` | Hostname label for the primary endpoint. | string | no |
| `spec.ipAddress` | Private IP address for the primary endpoint. | string | no |
| `spec.port` | Primary MySQL port. | int | no |
| `spec.portX` | X Plugin port. | int | no |
| `spec.backupPolicy` | Backup policy details. | object | no |
| `spec.source` | Restore source details. | object | no |
| `spec.maintenance` | Maintenance window settings. | object | no |
| `spec.freeformTags` | Free-form OCI tags. | object | no |
| `spec.definedTags` | Defined OCI tags. | object | no |
| `spec.deletionPolicy` | Delete protection and final backup settings. | object | no |
| `spec.crashRecovery` | Crash recovery mode. | string | no |
| `spec.databaseManagement` | Database Management service mode. | string | no |
| `spec.secureConnections` | TLS certificate configuration. | object | no |

When `spec.source` is set, choose a `spec.source.sourceType` and provide only
the matching variant field:

- `BACKUP`: `spec.source.backupId`
- `PITR`: `spec.source.dbSystemId` with optional `spec.source.recoveryPoint`
- `IMPORTURL`: `spec.source.sourceUrl`
- `NONE`: no additional source field

## MySQL DbSystem Status Parameters

| Parameter | Description | Type |
| --- | --- | --- |
| `status.status.conditions.type` | OSOK lifecycle condition such as provisioning, active, failed, or terminating. | string |
| `status.status.conditions.message` | Condition message. | string |
| `status.status.conditions.reason` | Condition reason. | string |
| `status.status.ocid` | OCI identifier tracked by OSOK. | string |
| `status.id` | OCI DB System OCID returned by the generated runtime. | string |
| `status.displayName` | Observed display name. | string |
| `status.lifecycleState` | Observed OCI lifecycle state. | string |
| `status.timeCreated` | OCI creation time. | string |
| `status.timeUpdated` | OCI update time. | string |
| `status.currentPlacement` | Observed availability and fault domain placement. | object |
| `status.adminUsername.secret.secretName` | Last applied administrative username secret reference. | string |
| `status.adminPassword.secret.secretName` | Last applied administrative password secret reference. | string |

## Provisioning a MySQL DB System

Provisioning a MySQL DB System requires a same-namespace Kubernetes Secret for
the administrative credentials. The operator reads the `username` and
`password` keys from that Secret when it builds the OCI create request; the
credential values are not stored in the CR.

- `SUBNET_OCID`: OCID of the subnet created in the pre-requisites step
- `CONFIGURATION_OCID`: OCID of the MySQL configuration to attach

Create the credentials Secret first:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: admin-secret
type: Opaque
stringData:
  username: admin
  password: S3cr3t!
```

```yaml
apiVersion: mysql.oracle.com/v1beta1
kind: DbSystem
metadata:
  name: example-dbsystem
spec:
  compartmentId: <COMPARTMENT_OCID>
  displayName: <DISPLAY_NAME>
  shapeName: <SHAPE>
  subnetId: <SUBNET_OCID>
  configurationId: <CONFIGURATION_OCID>
  availabilityDomain: <AVAILABILITY_DOMAIN>
  adminUsername:
    secret:
      secretName: admin-secret
  adminPassword:
    secret:
      secretName: admin-secret
  description: <DESCRIPTION>
  dataStorageSizeInGBs: <DB_SIZE>
  port: <PORT>
  portX: <PORTX>
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAG_NAMESPACE>:
      <KEY1>: <VALUE1>
```

Apply the resource:

```sh
kubectl apply -f <CREATE_YAML>.yaml
```

Inspect the resource:

```sh
kubectl get dbsystems
kubectl get dbsystems -o wide
kubectl describe dbsystems <NAME_OF_CR_OBJECT>
```

## Updating a MySQL DB System

Update the same `DbSystem` object by modifying supported mutable fields. Keep
the admin credential references in their secret-backed form.

```yaml
apiVersion: mysql.oracle.com/v1beta1
kind: DbSystem
metadata:
  name: example-dbsystem
spec:
  compartmentId: <COMPARTMENT_OCID>
  shapeName: <SHAPE>
  subnetId: <SUBNET_OCID>
  displayName: <UPDATED_DISPLAY_NAME>
  description: <UPDATED_DESCRIPTION>
  configurationId: <UPDATED_CONFIGURATION_OCID>
  adminUsername:
    secret:
      secretName: admin-secret
  adminPassword:
    secret:
      secretName: admin-secret
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAG_NAMESPACE>:
      <KEY1>: <VALUE1>
```

Apply the updated manifest:

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

## Kubernetes Secrets

The generated v2 mysql `DbSystem` surface preserves secret-backed admin
credentials without restoring the old handwritten runtime path.

- `spec.adminUsername.secret.secretName` must reference a Secret in the same
  namespace with a `username` entry.
- `spec.adminPassword.secret.secretName` must reference a Secret in the same
  namespace with a `password` entry.
- If either admin secret reference is omitted, OSOK omits that field from the
  OCI request instead of sending an empty string.
- OSOK mirrors only non-empty referenced secret names into
  `status.adminUsername` and `status.adminPassword` for drift tracking, but it
  does not write the secret payload into the CR status.
- OSOK does not create a follow-up Kubernetes Secret containing DB System
  connection details for this generated surface.
