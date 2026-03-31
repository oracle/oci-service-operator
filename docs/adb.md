# Oracle Autonomous Database Service

- [Introduction](#introduction)
- [OCI Permission requirement](#oci-permission-requirement)
- [Autonomous Database specification parameters](#autonomous-database-specification-parameters)
- [Autonomous Database status parameters](#autonomous-database-status-parameters)
- [Provision](#provisioning-an-autonomous-database)
- [Bind](#binding-and-wallet-access)
- [Update](#updating-an-autonomous-database)
- [Access information](#access-information)

## Introduction

[Oracle Autonomous Database Service](https://docs.oracle.com/en-us/iaas/Content/Database/Concepts/adboverview.htm)
is a fully managed, preconfigured database environment. It delivers automated
patching, upgrades, and tuning while remaining available to applications.

The v2 rollout treats the generated Autonomous Database surfaces under
`api/database/v1beta1` as the source of truth. The published OSOK CR kind
remains `AutonomousDatabases` for compatibility even though the underlying OCI
resource family is `AutonomousDatabase`. The current published spec still
includes `spec.id` for binding to an existing Autonomous Database and
`spec.wallet` for wallet-secret generation on the primary CR.

## OCI Permission requirement

**For instance principal**
The OCI Service Operator dynamic group should have the `manage` permission for
the `autonomous-database` resource type.

**Sample policy:**

```plain
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to manage autonomous-database in compartment <COMPARTMENT_NAME>
```

**For user principal**
The OCI Service Operator user should have the `manage` permission for the
`autonomous-database` resource type.

**Sample policy:**

```plain
Allow group <OSOK_GROUP> to manage autonomous-database in compartment <COMPARTMENT_NAME>
```

## Autonomous Database specification parameters

The checked-in type definitions under `api/database/v1beta1/` are the source of
truth for the full schema. Commonly used fields include:

| Parameter | Description | Type | Mandatory |
| --- | --- | --- | --- |
| `spec.id` | Existing Autonomous Database OCID. Set this to bind the CR to a pre-existing database instead of provisioning a new one. | string | no |
| `spec.compartmentId` | Compartment OCID for the Autonomous Database. | string | yes |
| `spec.displayName` | Friendly name for the database. | string | no |
| `spec.dbName` | Database name. | string | no |
| `spec.dbWorkload` | Workload type such as `OLTP`, `DW`, `AJD`, or `APEX`. | string | no |
| `spec.cpuCoreCount` / `spec.computeCount` | Requested compute. Use the OCI-supported combination for the selected compute model. | number | no |
| `spec.dataStorageSizeInTBs` / `spec.dataStorageSizeInGBs` | Requested storage capacity. | number | no |
| `spec.adminPassword.secret.secretName` | Kubernetes Secret that stores the admin password under the `password` key. | string | conditional |
| `spec.secretId` | OCI Vault secret OCID to use instead of a Kubernetes Secret. | string | conditional |
| `spec.wallet.walletName` | Kubernetes Secret name that should receive generated wallet files. Defaults to `<metadata.name>-wallet` when omitted. | string | no |
| `spec.wallet.walletPassword.secret.secretName` | Kubernetes Secret that stores the wallet password under the `walletPassword` key. When set, the current runtime generates a wallet Secret during create or bind. | string | no |
| `spec.kmsKeyId`, `spec.vaultId`, `spec.subnetId`, `spec.nsgIds` | OCI integration fields exposed directly by the generated schema. | mixed | no |
| `spec.freeformTags`, `spec.definedTags` | OCI tagging support. | map | no |

## Autonomous Database status parameters

| Parameter | Description | Type |
| --- | --- | --- |
| `status.status.ocid` | OSOK-tracked OCI identifier for the managed database. | string |
| `status.lifecycleState` | OCI lifecycle state projected from the runtime. | string |
| `status.id` | Autonomous Database OCID returned by OCI. | string |
| `status.connectionStrings` / `status.connectionUrls` | OCI connection metadata returned on the status surface. | object |
| `status.freeformTags`, `status.definedTags`, `status.systemTags` | OCI tag projections. | map |
| `status.status.conditions[]` | Shared OSOK condition history. | array |

## Provisioning an Autonomous Database

Store the admin password in a Kubernetes Secret under the `password` key before
applying the CR, or use `secretId` if OCI Vault should supply the password
material.

```sh
kubectl create secret generic <ADMIN_PASSWORD_SECRET_NAME> --from-literal=password=<ADMIN_PASSWORD>
```

If you keep the optional `spec.wallet` stanza shown below, create a wallet
password Secret under the `walletPassword` key before applying the CR:

```sh
kubectl create secret generic <WALLET_PASSWORD_SECRET_NAME> --from-literal=walletPassword=<WALLET_PASSWORD>
```

```yaml
apiVersion: database.oracle.com/v1beta1
kind: AutonomousDatabases
metadata:
  name: autonomousdatabases-sample
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
  wallet:
    walletName: <WALLET_SECRET_NAME>
    walletPassword:
      secret:
        secretName: <WALLET_PASSWORD_SECRET_NAME>
```

Apply the `AutonomousDatabases` resource with:

```sh
kubectl apply -f <CREATE_YAML>.yaml
```

## Binding and wallet access

The published `AutonomousDatabases` CR still supports the compatibility bind
flow through `spec.id`. Set it to the Autonomous Database OCID when you want
OSOK to bind to an existing database instead of provisioning a new one.

When `spec.wallet.walletPassword.secret.secretName` is set, the current runtime
generates a wallet Secret as part of create or bind. The Secret name comes from
`spec.wallet.walletName`; if that field is omitted, OSOK defaults it to
`<metadata.name>-wallet`.

Dedicated wallet resources remain available for standalone follow-on wallet
workflows:

- `config/samples/database_v1beta1_autonomousdatabasewallet.yaml`
- `config/samples/database_v1beta1_autonomousdatabaseregionalwallet.yaml`

## Updating an Autonomous Database

Update OCI-supported mutable fields on the generated v2 spec and reapply the
resource. For example:

```yaml
apiVersion: database.oracle.com/v1beta1
kind: AutonomousDatabases
metadata:
  name: <CR_OBJECT_NAME>
spec:
  displayName: <DISPLAY_NAME>
  computeCount: <COUNT>
  isAutoScalingEnabled: <true/false>
  freeformTags:
    <KEY1>: <VALUE1>
```

Apply the update with:

```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

## Access information

Use `status.connectionStrings` and `status.connectionUrls` on the primary CR
for connection metadata.

If `spec.wallet.walletPassword` is set, the current service manager also
materializes wallet files into a Kubernetes Secret during create or bind. The
target Secret name is `spec.wallet.walletName`, or `<metadata.name>-wallet`
when `walletName` is omitted.

Use the dedicated wallet resources when you need standalone or regional wallet
workflows that are separate from the primary `AutonomousDatabases` lifecycle.
