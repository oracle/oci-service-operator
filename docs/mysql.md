# Oracle MySQL Database Service

- [Introduction](#introduction)
- [MySQL DB System Pre-requisites](#pre-requisites-for-setting-up-mysql-db-systems)
- [MySQL DB System Specification Parameters](#mysql-db-system-specification-parameters)
- [MySQL DB System Status Parameters](#mysql-db-system-status-parameters)
- [Provision](#provisioning-a-mysql-db-system)
- [Current v2 notes](#current-v2-notes)
- [Access Information](#access-information)

## Introduction

[Oracle MySQL Database Service](https://www.oracle.com/mysql/) is a fully managed database service that lets developers quickly develop and deploy secure, cloud native applications using the world’s most popular open source database. Oracle MySQL Database Service is also offered via the OCI Service Operator for Kubernetes, making it easy for applications to provision and integrate seamlessly with MySQL databases.


## Pre-requisites for setting up MySQL DB Systems

If this is your first time using MySQL Database Service, ensure your tenancy administrator has performed the following tasks:

### Create VCN/Subnets
  - [Virtual Networking Quickstart](https://docs.oracle.com/en-us/iaas/Content/Network/Tasks/quickstartnetworking.htm) Create VCN and subnets using Virtual Cloud Networks > Start VCN Wizard > Create a VCN with Internet Connectivity.
  - It would be advisable if the VCN for Mysql DbSystem is created in the same vcn as the Kubernetes cluster.
  - [Comprehensive networking setup](https://docs.oracle.com/en-us/iaas/mysql-database/doc/networking-setup-mysql-db-systems.html#MYAAS-GUID-2B4F78DD-72D3-45BA-8F6A-AC5E3A11B729)

### Create Policies

Create policies in the root compartment with the following statements [Policy Setup Documentation](https://docs.oracle.com/en-us/iaas/mysql-database/doc/policy-details-mysql-database-service.html#GUID-2D9D3C84-07A3-4BEE-82C7-B5A72A943F53)

**When using Instance Principals**
The OCI Service Operator dynamic group should have the `manage` permission for the `mysql-family` resource type. Use this approach when levraging Instance Principals for OSOK. This the recommended approach for running OSOK within OCI.

**Sample Policy:**

```plain
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to {SUBNET_READ, SUBNET_ATTACH, SUBNET_DETACH, VCN_READ, COMPARTMENT_INSPECT} in [ tenancy | compartment <compartment_name> | compartment id <compartment_ocid> ]
```
```plain
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to manage mysql-family in [ tenancy | compartment <compartment_name> | compartment id <compartment_ocid> ]
```
```plain
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to use tag-namespaces in tenancy
```

**When using User Principals**
The OCI Service Operator user should have the `manage` permission for the `mysql-family` resource type. Use this approach when levraging User Principals for OSOK. This the recommended approach for running OSOK outside OCI.


**Sample Policy:**

```plain
Allow group <OSOK_GROUP> to {SUBNET_READ, SUBNET_ATTACH, SUBNET_DETACH, VCN_READ, COMPARTMENT_INSPECT} in [ tenancy | compartment <compartment_name> | compartment id <compartment_ocid> ]
```
```plain
Allow group <OSOK_GROUP> to manage mysql-family in [ tenancy | compartment <compartment_name> | compartment id <compartment_ocid> ]
```
```plain
Allow group <OSOK_GROUP> to use tag-namespaces in tenancy
```


Without these policies, the service will not function correctly.

## MySQL DB System Specification Parameters

The generated v2 `DbSystem` CR is defined in
`api/mysql/v1beta1/dbsystem_types.go`. Commonly used spec fields are
summarized below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.displayName` | The user-friendly name for the DB System. | string | no |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the Mysql DbSystem. | string | yes       |
| `spec.shapeName` | The name of the shape. The shape determines the resources allocated. CPU cores and memory for VM shapes; CPU cores, memory and storage for non-VM (or bare metal) shapes.  | string | yes       |
| `spec.subnetId` | The OCID of the subnet the DB System is associated with.  | string | yes       |
| `spec.dataStorageSizeInGBs`| Initial size of the data volume in GBs that will be created and attached. Keep in mind that this only specifies the size of the database data volume, the log volume for the database will be scaled appropriately with its shape. | int    | no |
| `spec.isHighlyAvailable` | Specifies if the DB System is highly available.  | boolean | no |
| `spec.availabilityDomain`| The availability domain on which to deploy the Read/Write endpoint. This defines the preferred primary instance. | string | no |
| `spec.faultDomain`| The fault domain on which to deploy the Read/Write endpoint. This defines the preferred primary instance. | string | no        |
| `spec.configurationId` | The OCID of the Configuration to be used for this DB System. [More info about Configurations](https://docs.oracle.com/en-us/iaas/mysql-database/doc/db-systems.html#GUID-E2A83218-9700-4A49-B55D-987867D81871)| string | no |
| `spec.description` | User-provided data about the DB System. | string | no |
| `spec.hostnameLabel` | The hostname for the primary endpoint of the DB System. Used for DNS. | string | no |
| `spec.mysqlVersion` | The specific MySQL version identifier. | string | no |
| `spec.port` | The port for primary endpoint of the DB System to listen on. | int | no |
| `spec.portX` | The TCP network port on which X Plugin listens for connections. This is the X Plugin equivalent of port. | int | no |
| `spec.ipAddress` | The IP address the DB System is configured to listen on. A private IP address of your choice to assign to the primary endpoint of the DB System. Must be an available IP address within the subnet's CIDR. If you don't specify a value, Oracle automatically assigns a private IP address from the subnet. This should be a "dotted-quad" style IPv4 address. | string | no |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | string | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). | string | no |
| `spec.adminUsername` | Same-namespace Kubernetes Secret reference for the administrative username. OSOK reads the `username` key from the referenced Secret. | object | no |
| `spec.adminPassword` | Same-namespace Kubernetes Secret reference for the administrative password. OSOK reads the `password` key from the referenced Secret. | object | no |
| `spec.backupPolicy` | Nested backup policy options for automatic backup behavior and PITR. | object | no |
| `spec.maintenance` | Nested maintenance window configuration. | object | no |
| `spec.secureConnections` | Nested TLS certificate configuration. | object | no |

The generated v2 runtime resolves `spec.adminUsername.secret.secretName` and
`spec.adminPassword.secret.secretName` from Secrets in the same namespace as
the `DbSystem` CR. It sends the resolved `username` and `password` values to
OCI and mirrors only the secret references back into status.



## MySQL DB System Status Parameters

| Parameter                                         | Description                                                         | Type   | Mandatory |
| --------------------------------------------------| ------------------------------------------------------------------- | ------ | --------- |
| `status.osokstatus.conditions.type`               | Lifecycle state of the Mysql DbSystem Service. The following values are valid: <ul><li>**Provisioning** - indicates an Mysql DbSystem is provisioning. </li><li>**Active** - indicates an Mysql DbSystem is Active. </li><li>**Failed** - indicates an Mysql DbSystem failed provisioning. </li><li>**Terminating** - indicates an Mysql DbSystem is Deleting. </li></ul>|  string  |  no  |
| `status.osokstatus.conditions.status`             | Status of the Mysql DbSystem Custom Resource during the condition update. |  string  |  no  |
| `status.osokstatus.conditions.lastTransitionTime` | Last time the Mysql DbSystem  CR was Updated. |  string  |  no  | 
| `status.osokstatus.conditions.message`            | Message of the status condition of the CR. | string | no | 
| `status.osokstatus.conditions.reason`             | Resource if any of status condition of the CR. | string | no |
| `status.osokstatus.ocid`                          | The Mysql DbSystem [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm). |  string  | yes |
| `status.osokstatus.message`                       | Overall status message of the CR.  |  string  | no  |
| `status.osokstatus.reason`                        | Overall status reason of the CR.   | string | no |
| `status.osokstatus.createdAt`                     | Created time of the Mysql DbSystem.            | string | no |  
| `status.osokstatus.updatedAt`                     | Updated time of the Mysql DbSystem.            | string | no |
| `status.osokstatus.requestedAt`                   | Requested time of the CR.          | string | no |
| `status.osokstatus.deletedAt`                     | Deleted time of the CR.            | string | no | 

## Provisioning a MySQL DB System

The v2 generated runtime now uses the `DbSystem` kind directly. Use the current
API group and spec surface instead of the retired `MySqlDbSystem` resource and
its retired bind workflow.

- SUBNET_OCID - OCID of the subnet created in the pre-requisites step
- CONFIGURATION_ID - [More info about Configurations](https://docs.oracle.com/en-us/iaas/mysql-database/doc/db-systems.html#GUID-E2A83218-9700-4A49-B55D-987867D81871) Get your [Configuration_id](https://console.us-ashburn-1.oraclecloud.com/mysqlaas/configurations) 


```yaml
apiVersion: v1
kind: Secret
metadata:
  name: admin-secret
type: Opaque
stringData:
  username: <ADMIN_USERNAME>
  password: <ADMIN_PASSWORD>
---
apiVersion: mysql.oracle.com/v1beta1
kind: DbSystem
metadata:
  name: <CR_OBJECT_NAME>
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
  isHighlyAvailable: false
```

Create the Secret in the same namespace as the `DbSystem` CR. OSOK reads the
`username` and `password` keys from that Secret during reconcile.

Run the following command to create a CR to the cluster:
```sh
kubectl apply -f <CREATE_YAML>.yaml
```

Once the CR is created, OSOK reconciles the `DbSystem` and records the OCI
identifier in status.

`status.adminUsername.secret.secretName` and
`status.adminPassword.secret.secretName` mirror the last applied secret
references. Secret values are not copied into status.

The `DbSystem` CR can list the MySQL DB Systems in the cluster:
```sh
$ kubectl get dbsystems
NAME                       STATUS         AGE
dbsystem-sample            Active         4d
```

The `DbSystem` CR can list the MySQL DB Systems in the cluster with detailed information:
```sh
$ kubectl get dbsystems -o wide
NAME                         DISPLAYNAME     STATUS         OCID                                   AGE
dbsystem-sample              BusyBoxDB       Active         ocid1.dbsystem.oc1.iad.........        4d
```

The `DbSystem` CR can be described as below:
```sh
$ kubectl describe dbsystems <NAME_OF_CR_OBJECT>
```

## Current v2 notes

- The generated v2 contract no longer uses the legacy `MySqlDbSystem` kind.
- The generated runtime reads same-namespace Kubernetes Secret references for
  admin credentials but does not materialize endpoint or credential Secrets.
- Check `api/mysql/v1beta1/dbsystem_types.go` for the current spec surface when
  authoring CRs.

## Access Information

The generated v2 `DbSystem` runtime does not preserve the retired
secret-materialization flow from the handwritten `MySqlDbSystem` path. Consume
the current status surface on the CR directly:

- `status.ipAddress`
- `status.hostnameLabel`
- `status.port`
- `status.portX`
- `status.endpoints`
- `status.adminUsername.secret.secretName`
- `status.adminPassword.secret.secretName`
 
