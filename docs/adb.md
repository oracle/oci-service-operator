# Oracle Autonomous Database Service

- [Introduction](#introduction)
- [OCI Permission requirement](#oci-permission-requirement)
- [Autonomous Database Specification Parameters](#autonomous-database-specification-parameters)
- [Autonomous Database Status Parameters](#autonomous-database-status-parameters)
- [Provisioning an Autonomous Database](#provisioning-an-autonomous-database)
- [Binding to an Existing Autonomous Database](#binding-to-an-existing-autonomous-database)
- [Updating an Autonomous Database](#updating-an-autonomous-database)
- [Access Information in Kubernetes Secrets](#access-information-in-kubernetes-secrets)

## Introduction

[Oracle Autonomous Database Service](https://docs.oracle.com/en-us/iaas/Content/Database/Concepts/adboverview.htm) is a fully managed, preconfigured database environment. It delivers automated patching, upgrades, and tuning, including performing all routine database maintenance tasks while the system is running, without human intervention. Autonomous Database service is also offered via the OCI Service Operator for Kubernetes (OSOK), making it easy for applications to provision and integrate seamlessly.

`database.oracle.com/v1beta1` now uses the generated v2
`AutonomousDatabase` surface. The legacy handwritten
`AutonomousDatabases` compatibility resource, its custom runtime behavior, and
its manual webhook seam are removed.

## OCI Permission requirement

**For Instance Principle**
The OCI Service Operator dynamic group should have the `manage` permission for the `autonomous-database` resource type.

**Sample Policy:**

```plain
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to manage autonomous-database in compartment <COMPARTMENT_NAME>
```

**For User Principle**
The OCI Service Operator user should have the `manage` permission for the `autonomous-database` resource type.

**Sample Policy:**

```plain
Allow group <OSOK_GROUP> to manage autonomous-database in compartment <COMPARTMENT_NAME>
```

## Autonomous Database Specification Parameters

The generated v2 `AutonomousDatabase` CR exposes a much larger OCI-aligned
surface than the legacy handwritten resource. Common fields are summarized
below; the checked-in type definition in
`api/database/v1beta1/autonomousdatabase_types.go` is the source of truth for
the full schema.

| Parameter | Description | Type | Mandatory |
| --- | --- | --- | --- |
| `spec.compartmentId` | The compartment OCID for the Autonomous Database. | string | yes |
| `spec.displayName` | Friendly name for the Autonomous Database. | string | no |
| `spec.dbName` | Database name. | string | no |
| `spec.dbWorkload` | Workload type such as `OLTP`, `DW`, `AJD`, or `APEX`. | string | no |
| `spec.cpuCoreCount` / `spec.computeCount` | Requested compute. Use the OCI-supported combination for the selected compute model. | number | no |
| `spec.dataStorageSizeInTBs` / `spec.dataStorageSizeInGBs` | Requested storage capacity. | number | no |
| `spec.adminPassword.secret.secretName` | Kubernetes Secret name that stores the admin password under the `password` key. | string | conditional |
| `spec.secretId` | OCI Vault secret OCID alternative to `spec.adminPassword.secret.secretName`. | string | conditional |
| `spec.kmsKeyId`, `spec.vaultId`, `spec.subnetId`, `spec.nsgIds` | OCI integration fields exposed directly by the generated schema. | mixed | no |
| `spec.freeformTags`, `spec.definedTags` | OCI tagging support. | map | no |

The v2 resource continues to accept Secret-backed
`spec.adminPassword.secret.secretName` for the admin credential and also
supports `spec.secretId` for OCI Vault-backed password material. It no longer
accepts `kind: AutonomousDatabases`, `spec.wallet`, or `spec.walletPassword`.

## Autonomous Database Status Parameters

| Parameter | Description | Type |
| --- | --- | --- |
| `status.status.ocid` | OSOK-tracked OCI identifier for the managed database. | string |
| `status.lifecycleState` | OCI lifecycle state projected from the generated runtime. | string |
| `status.id` | Autonomous Database OCID returned by OCI. | string |
| `status.connectionStrings` / `status.connectionUrls` | OCI connection metadata returned on the status surface. | object |
| `status.freeformTags`, `status.definedTags`, `status.systemTags` | OCI tag projections. | map |
| `status.status.conditions[]` | Shared OSOK condition history. | array |

## Provisioning an Autonomous Database

The generator-owned `AutonomousDatabase` controller provisions an Autonomous
Database directly from the v2 spec fields. The legacy webhook and compatibility
runtime are gone, so manifests should use only the generated field names.

The following example shows a typical create flow. Store the admin password in
a Kubernetes Secret under the `password` key before applying the CR, or use
`secretId` instead if you want OCI Vault to provide the password material.

```sh
kubectl create secret generic <ADMIN_PASSWORD_SECRET_NAME> --from-literal=password=<ADMIN_PASSWORD>
```

```yaml
apiVersion: database.oracle.com/v1beta1
kind: AutonomousDatabase
metadata:
  name: autonomousdatabase-sample
spec:
  compartmentId: <COMPARTMENT_OCID>
  displayName: <DISPLAY_NAME>
  dbName: <DB_NAME>
  dbWorkload: OLTP
  dbVersion: <ORACLE_DB_VERSION>
  dataStorageSizeInTBs: <SIZE_IN_TBS>
  cpuCoreCount: <COUNT>
  adminPassword:
    secret:
      secretName: <ADMIN_PASSWORD_SECRET_NAME>
  isAutoScalingEnabled: <true/false>
  isFreeTier: <false/true>
  licenseModel: <BRING_YOUR_OWN_LICENSE/LICENSE_INCLUDED>
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAGNAMESPACE1>:
      <KEY1>: <VALUE1>
```

Run the following command to create a CR in the cluster:
```sh
kubectl apply -f <CREATE_YAML>.yaml
```

Once the CR is created, OSOK reconciles it through the generated runtime path
and projects the OCI lifecycle into the CR status.

## Binding to an Existing Autonomous Database

The old `AutonomousDatabases` compatibility resource supported a handwritten
bind-and-wallet shortcut. That shortcut is removed with the generator-only v2
rollout. Use the generated `AutonomousDatabase` resource for lifecycle
management, and use the OCI identifiers projected into status together with the
dedicated wallet resources for follow-on access workflows.

## Updating an Autonomous Database

Update the existing `AutonomousDatabase` CR by changing OCI-supported mutable
fields on the generated v2 spec. For example:

```yaml
apiVersion: database.oracle.com/v1beta1
kind: AutonomousDatabase
metadata:
  name: <CR_OBJECT_NAME>
spec:
  displayName: <DISPLAY_NAME>
  computeCount: <COUNT>
  isAutoScalingEnabled: <true/false>
  freeformTags:
    <KEY1>: <VALUE1>
```

Run the following command to create a CR that updates an existing Autonomous Database instance:
```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

## Access Information in Kubernetes Secrets

The legacy handwritten `AutonomousDatabases` runtime used to download wallet
material into a Kubernetes Secret as a side effect of create or bind. The
generated v2 `AutonomousDatabase` runtime no longer does that.

If you need wallet artifacts, use the dedicated generated resources:

- `config/samples/database_v1beta1_autonomousdatabasewallet.yaml`
- `config/samples/database_v1beta1_autonomousdatabaseregionalwallet.yaml`

Those resources own wallet retrieval explicitly. The primary
`AutonomousDatabase` resource now stays on the generator-owned OCI lifecycle
path only.
