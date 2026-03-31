# Oracle Autonomous Database Service

- [Introduction](#introduction)
- [OCI Permission requirement](#oci-permission-requirement)
- [Autonomous Database Specification Parameters](#autonomous-database-specification-parameters)
- [Autonomous Database Status Parameters](#autonomous-database-status-parameters)
- [Provision](#provisioning-an-autonomous-database)
- [Current v2 notes](#current-v2-notes)
- [Access Information](#access-information)

## Introduction

[Oracle Autonomous Database Service](https://docs.oracle.com/en-us/iaas/Content/Database/Concepts/adboverview.htm) is a fully managed, preconfigured database environment. It delivers automated patching, upgrades, and tuning, including performing all routine database maintenance tasks while the system is running, without human intervention. Autonomous Database service is also offered via the OCI Service Operator for Kubernetes (OSOK), making it easy for applications to provision and integrate seamlessly.

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

The generated v2 `AutonomousDatabase` CR is defined in
`api/database/v1beta1/autonomousdatabase_types.go`. Commonly used spec fields
are summarized below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.displayName` | The user-friendly name for the Autonomous Database. | string | no |
| `spec.dbName` | The database name. The name must begin with an alphabetic character and can contain up to 30 alphanumeric characters. | string | no |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the Autonomous Database. | string | yes       |
| `spec.computeModel` | The compute model for the database. `ECPU` is the recommended current model. | string | no |
| `spec.computeCount` | The compute amount to use with `spec.computeModel`. | float32 | no |
| `spec.cpuCoreCount` | Legacy OCPU-based sizing for the database. Use this instead of `spec.computeModel` and `spec.computeCount`, not in addition to them. | int | no |
| `spec.dataStorageSizeInTBs`| The size, in terabytes, of the data volume that will be created and attached to the database. | int | no |
| `spec.dbVersion` | A valid Oracle Database version for Autonomous Database. | string | no        |
| `spec.isDedicated` | True if the database is on dedicated [Exadata infrastructure](https://docs.cloud.oracle.com/Content/Database/Concepts/adbddoverview.htm).  | boolean | no       |
| `spec.dbWorkload`  | The Autonomous Database workload type. Common values include `OLTP`, `DW`, `AJD`, and `APEX`. | string | no |
| `spec.isAutoScalingEnabled`| Indicates if auto scaling is enabled for the Autonomous Database OCPU core count. The default value is `FALSE`. | boolean| no        |
| `spec.isFreeTier` | Indicates if this is an Always Free resource. The default value is false. Note that Always Free Autonomous Databases have 1 CPU and 20GB of memory. For Always Free databases, memory and CPU cannot be scaled. | boolean | no |
| `spec.licenseModel` | The Oracle license model that applies to the Oracle Autonomous Database. Bring your own license (BYOL) allows you to apply your current on-premises Oracle software licenses to equivalent, highly automated Oracle PaaS and IaaS services in the cloud. License Included allows you to subscribe to new Oracle Database software licenses and the Database service. Note that when provisioning an Autonomous Database on [dedicated Exadata infrastructure](https://docs.oracle.com/iaas/Content/Database/Concepts/adbddoverview.htm), this attribute must be null because the attribute is already set at the Autonomous Exadata Infrastructure level. When using [shared Exadata infrastructure](https://docs.oracle.com/iaas/Content/Database/Concepts/adboverview.htm#AEI), if a value is not specified, the system will supply the value of `BRING_YOUR_OWN_LICENSE`. <br>Allowed values are:<ul><li>LICENSE_INCLUDED</li><li>BRING_YOUR_OWN_LICENSE</li></ul>. | string | no       |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | string | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). | string | no |
| `spec.adminPassword` | The admin password to send directly in the CR. It must meet the Autonomous Database password policy. | string | conditionally required |
| `spec.secretId` | The OCI Vault secret OCID to use instead of `spec.adminPassword`. | string | conditionally required |
| `spec.secretVersionNumber` | The OCI Vault secret version to use with `spec.secretId`. | int | no |
| `spec.subnetId` | The private subnet OCID for private endpoint databases. | string | no |
| `spec.nsgIds` | Network security groups attached to the private endpoint. | []string | no |
| `spec.privateEndpointLabel` | The private endpoint label for private access. | string | no |
| `spec.privateEndpointIp` | The private endpoint IP for private access. | string | no |

Provide either `spec.adminPassword` or `spec.secretId` when the service
requires credentials. The generated v2 runtime does not read Kubernetes Secret
references for password or wallet materialization.

## Autonomous Database Status Parameters

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

## Provisioning an Autonomous Database

The v2 generated runtime now uses the `AutonomousDatabase` kind directly. Use
the current API group and spec surface instead of the retired
`AutonomousDatabases` resource and its retired wallet-download and bind helper
workflow.

Set credentials on the CR itself with `spec.adminPassword`, or reference an OCI
Vault secret with `spec.secretId` and `spec.secretVersionNumber`.

```yaml
apiVersion: database.oracle.com/v1beta1
kind: AutonomousDatabase
metadata:
  name: <CR_OBJECT_NAME>
spec:
  compartmentId: <COMPARTMENT_OCID>
  displayName: <DISPLAY_NAME>
  dbName: <DB_NAME>
  dbWorkload: OLTP
  computeModel: ECPU
  computeCount: 2
  dataStorageSizeInTBs: <SIZE_IN_TBs>
  adminPassword: <ADMIN_PASSWORD>
```

Run the following command to create a CR in the cluster:

```sh
kubectl apply -f <CREATE_YAML>.yaml
```

Once the CR is created, OSOK reconciles the `AutonomousDatabase` and records
the OCI identifier in status.

List and inspect the resource with:

```sh
kubectl get autonomousdatabases
kubectl get autonomousdatabases -o wide
kubectl describe autonomousdatabases <NAME_OF_CR_OBJECT>
```

## Current v2 notes

- The generated v2 contract no longer uses the legacy `AutonomousDatabases`
  kind.
- The retired handwritten ADB runtime's nested Kubernetes Secret helper fields,
  wallet download examples, and bind-by-id examples do not apply to the
  generated v2 resource.
- Check `api/database/v1beta1/autonomousdatabase_types.go` for the current
  spec surface when authoring CRs.

## Access Information

The generated v2 `AutonomousDatabase` runtime does not preserve the retired
wallet-secret side effects from the handwritten `AutonomousDatabases` path.
Consume the current status surface on the CR directly:

- `status.serviceConsoleUrl`
- `status.connectionStrings`
- `status.connectionUrls`
- `status.privateEndpoint`
 
