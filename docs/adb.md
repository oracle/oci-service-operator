# Oracle Autonomous Database Service

- [Introduction](#introduction)
- [OCI Permission requirement](#oci-permission-requirement)
- [Autonomous Database Specification Parameters](#autonomous-database-specification-parameters)
- [Autonomous Database Status Parameters](#autonomous-database-status-parameters)
- [Provision](#provisioning-an-adb)
- [Bind](#binding-to-an-existing-adb)
- [Update](#updating-an-adb)
- [Access Information in Kubernetes Secret](#access-information-in-kubernetes-secrets)

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

The Complete Specification of the `AutonomousDatabase` Custom Resource (CR) is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.id` | The Autonomous Database [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm). | string | no  |
| `spec.displayName` | The user-friendly name for the Autonomous Database. The name does not have to be unique. | string | yes       |
| `spec.dbName` | The database name. The name must begin with an alphabetic character and can contain a maximum of 14 alphanumeric characters. Special characters are not permitted. The database name must be unique in the tenancy. | string | yes       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the Autonomous Database. | string | yes       |
| `spec.cpuCoreCount` | The number of OCPU cores to be made available to the database. | int    | yes       |
| `spec.dataStorageSizeInTBs`| The size, in terabytes, of the data volume that will be created and attached to the database. This storage can later be scaled up if needed. | int    | yes       |
| `spec.dbVersion` | A valid Oracle Database version for Autonomous Database. | string | no        |
| `spec.isDedicated` | True if the database is on dedicated [Exadata infrastructure](https://docs.cloud.oracle.com/Content/Database/Concepts/adbddoverview.htm).  | boolean | no       |
| `spec.dbWorkload`  | The Autonomous Database workload type. The following values are valid:  <ul><li>**OLTP** - indicates an Autonomous Transaction Processing database</li><li>**DW** - indicates an Autonomous Data Warehouse database</li></ul>  | string | yes       |
| `spec.isAutoScalingEnabled`| Indicates if auto scaling is enabled for the Autonomous Database OCPU core count. The default value is `FALSE`. | boolean| no        |
| `spec.isFreeTier` | Indicates if this is an Always Free resource. The default value is false. Note that Always Free Autonomous Databases have 1 CPU and 20GB of memory. For Always Free databases, memory and CPU cannot be scaled. | boolean | no |
| `spec.licenseModel` | The Oracle license model that applies to the Oracle Autonomous Database. Bring your own license (BYOL) allows you to apply your current on-premises Oracle software licenses to equivalent, highly automated Oracle PaaS and IaaS services in the cloud. License Included allows you to subscribe to new Oracle Database software licenses and the Database service. Note that when provisioning an Autonomous Database on [dedicated Exadata infrastructure](https://docs.oracle.com/iaas/Content/Database/Concepts/adbddoverview.htm), this attribute must be null because the attribute is already set at the Autonomous Exadata Infrastructure level. When using [shared Exadata infrastructure](https://docs.oracle.com/iaas/Content/Database/Concepts/adboverview.htm#AEI), if a value is not specified, the system will supply the value of `BRING_YOUR_OWN_LICENSE`. <br>Allowed values are:<ul><li>LICENSE_INCLUDED</li><li>BRING_YOUR_OWN_LICENSE</li></ul>. | string | no       |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | string | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). | string | no |
| `spec.adminPassword.secret.secretName` | The Kubernetes Secret Name that contains admin password for Autonomous Database. The password must be between 12 and 30 characters long, and must contain at least 1 uppercase, 1 lowercase, and 1 numeric character. It cannot contain the double quote symbol (") or the username "admin", regardless of casing. | string | yes       |
| `spec.wallet.walletName` | The Kubernetes Secret Name of the wallet which contains the downloaded wallet information. | string | yes       |
| `spec.walletPassword.secret.secretName`| The Kubernetes Secret Name that contains the password to be used for downloading the Wallet. | string |  no  |

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

Provisioning of an Autonomous Database requires you to input the admin password as a Kubernetes secret. OSOK acquires the admin password from the Kubernetes secret provided in the `spec`. 
The Kubernetes secret should contain the admin password in `password` field. 
```sh
kubectl create secret generic <ADMIN-PASSWORD-SECRET-NAME> --from-literal=password=<ADMIN-PASSWORD>
```

The Autonomous Database can be accessed using the details in the wallet which will be downloaded as part of the provision/bind operation of the CR. OSOK acquires the wallet password from the Kubernetes secret whose name is provided in the `spec`. Also, we can configure the name of the wallet in the `spec`.

```sh
kubectl create secret generic <WALLET-PASSWORD-SECRET-NAME> --from-literal=walletpassword=<WALLET-PASSWORD>
```

The OSOK AutonomousDatabases controller automatically provisions an Autonomous Database when you provides mandatory fields to the `spec`. the following is a sample YAML for Autonomous Database.

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: AutonomousDatabases
metadata:
  name: <CR_OBJECT_NAME>
spec:
  compartmentId: <COMPARTMENT_OCID>
  displayName: <DISPLAY_NAME>
  dbName: <DB_NAME>
  dbWorkload: <OLTP/DW>
  isDedicated: <false/true>
  dbVersion: <ORABLE_DB_VERSION>
  dataStorageSizeInTBs: <SIZE_IN_TBs>
  cpuCoreCount: <COUNT>
  adminPassword:
    secret:
      secretName: <ADMIN_PASSWORD_SECRET_NAME>
  isAutoScalingEnabled: <true/false>
  isFreeTier: <false/true>
  licenseModel: <BRING_YOUR_OWN_LICENSE/LICENSE_INCLUDEE>
  wallet:
    walletName: <WALLET_SECRET_NAME>
    walletPassword:
      secret:
        secretName: <WALLET_PASSWORD_SECRET_NAME>
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

Once the CR is created, OSOK will reconcile and create an Autonomous Database. OSOK will ensure the Autonomous Database instance is available.

The AutonomousDatabases CR can list the Autonomous Databases in the cluster as below: 
```sh
$ kubectl get autonomousdatabases
NAME                         DBWORKLOAD   STATUS         AGE
autonomousdatabases-sample   OLTP         Active         4d
```

The AutonomousDatabases CR can list the Autonomous Databases in the cluster with detailed information as below: 
```sh
$ kubectl get autonomousdatabases -o wide
NAME                         DISPLAYNAME   DBWORKLOAD   STATUS         OCID                                   AGE
autonomousdatabases-sample   ADBTest       OLTP         Active         ocid1.autonomousdatabase.oc1........   4d
```

The AutonomousDatabases CR can be describe as below:
```sh
$ kubectl describe autonomousdatabases <NAME_OF_CR_OBJECT>
```

## Binding to an Existing Autonomous Database

The OSOK allows you to bind to an existing Autonomous Database instance. In this case, `Id` is the only required field in the CR `spec`. The wallet information can be provided to obtain the access information of the Autonomous Database instance.

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: AutonomousDatabases
metadata:
  name: <CR_OBJECT_NAME>
spec:
  id: <AUTONOMOUS_DATABASE_OCID>
  wallet:
    walletName: <WALLET_SECRET_NAME>
    walletPassword:
      secret:
        secretName: <WALLET_PASSWORD_SECRET_NAME>
```

Run the following command to create a CR that binds to an existing DB instance:
```sh
kubectl apply -f <BIND_YAML>.yaml
```

## Updating an Autonomous Database

Customers can also update a number of [parameters](https://docs.oracle.com/en-us/iaas/api/#/en/database/20160918/datatypes/UpdateAutonomousDatabaseDetails) of the Autonomous Database instance.
```yaml
apiVersion: oci.oracle.com/v1beta1
kind: AutonomousDatabases
metadata:
  name: <CR_OBJECT_NAME>
spec:
  id: <AUTONOMOUS_DATABASE_OCID>
  displayName: <DISPLAY_NAME>
  dbName: <DB_NAME>
  dbWorkload: <OLTP/DW>
  isDedicated: <false/true>
  dbVersion: <ORABLE_DB_VERSION>
  dataStorageSizeInTBs: <SIZE_IN_TBs>
  cpuCoreCount: <COUNT>
  adminPassword:
    secret:
      secretName: <ADMIN_PASSWORD_SECRET_NAME>
  isAutoScalingEnabled: <true/false>
  isFreeTier: <false/true>
  licenseModel: <BRING_YOUR_OWN_LICENSE/LICENSE_INCLUDEE>
  wallet:
    walletName: <WALLET_SECRET_NAME>
    walletPassword:
      secret:
        secretName: <WALLET_PASSWORD_SECRET_NAME>
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAGNAMESPACE1>:
      <KEY1>: <VALUE1>
```

Run the following command to create a CR that updates an existing Autonomous Database instance:
```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

## Access Information in Kubernetes Secrets

The Access information of a OCI Service or resource will be created as a Kubernetes secret to manage the Autonomous Database. The name of the secret can be provided in the CR yaml or by default the name of the CR will be used.

Customer will get the access information as Kubernetes secret to use the Autonomous Database. The following files/details will be made available to the user:

| Parameter          | Description                                                              | Type   |
| ------------------ | ------------------------------------------------------------------------ | ------ |
| `ewallet.p12`      | Oracle Wallet.                                                           | string |
| `cwallet.sso`      | Oracle wallet with autologin.                                            | string |
| `tnsnames.ora`     | Configuration file containing service name and other connection details. | string |
| `sqlnet.ora`       |                                                                          | string |
| `ojdbc.properties` |                                                                          | string |
| `keystore.jks`     | Java Keystore.                                                           | string |
| `truststore.jks`   | Java trustore.                                                           | string |
| `user_name`        | Pre-provisioned DB ADMIN Username.                                       | string |
 
