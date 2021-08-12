# Mysql DbSystem Services

- [Introduction](#introduction)
- [Mysql DbSystem Pre-requisites](#pre-requisites-for-setting-up-mysql-dbsystems)
- [Mysql DbSystem Specification Parameters](#mysql-dbsystem-specification-parameters)
- [Mysql DbSystem Status Parameters](#mysql-dbsystem-status-parameters)
- [Provision](#provisioning-a-mysql-dbsystem)
- [Bind](#binding-to-an-existing-mysql-dbsystem)
- [Update](#updating-a-mysql-dbsystem)
- [Access Information in Kubernetes Secret](#access-information-in-kubernetes-secrets)

## Introduction

[Oracle MySQL Database Service](https://www.oracle.com/mysql/) is a fully managed database service that lets developers quickly develop and deploy secure, cloud native applications using the world’s most popular open source database. Oracle MySQL Database Service is also offered via OCI Service Operator thereby making it easy for applications to provision and integrate seamlessly with Mysql databases.


## Pre-requisites for setting up Mysql DbSystems

If this is your first time using MySQL Database Service, ensure your tenancy administrator has performed the following tasks:

### Create VCN/Subnets
  - [Virtual Networking Quickstart](https://docs.oracle.com/en-us/iaas/Content/Network/Tasks/quickstartnetworking.htm) Create VCN and subnets using Virtual Cloud Networks > Start VCN Wizard > Create a VCN with Internet Connectivity.
  - It would be advisable if the VCN for Mysql DbSystem is created in the same vcn as the Kubernetes cluster.
  - [Comprehensive networking setup](https://docs.oracle.com/en-us/iaas/mysql-database/doc/networking-setup-mysql-db-systems.html#MYAAS-GUID-2B4F78DD-72D3-45BA-8F6A-AC5E3A11B729)

### Create Policies

Create policies in the root compartment with the following statements [Policy Setup Documentation](https://docs.oracle.com/en-us/iaas/mysql-database/doc/policy-details-mysql-database-service.html#GUID-2D9D3C84-07A3-4BEE-82C7-B5A72A943F53)

**For Instance Principle**
The Dynamic group for OCI Service Operator should have permission `manage` for resource type `mysql-family`.

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

**For User Principle**
The OCI OSOK user for OCI Service Operator should have permission `manage` for resource type `mysql-family`.

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

## Mysql DbSystem Specification Parameters

The Complete Specification of the `mysqldbsystems` Custom Resource is as detailed below:

| Parameter                          | Description                                                         | Type   | Mandatory |
| ---------------------------------- | ------------------------------------------------------------------- | ------ | --------- |
| `spec.id` | The Mysql DbSystem [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm). | string | no  |
| `spec.displayName` | The user-friendly name for the Mysql DbSystem. The name does not have to be unique. | string | yes       |
| `spec.compartmentId` | The [OCID](https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) of the compartment of the Mysql DbSystem. | string | yes       |
| `spec.shapeName` | The name of the shape. The shape determines the resources allocated. CPU cores and memory for VM shapes; CPU cores, memory and storage for non-VM (or bare metal) shapes.  | string | yes       |
| `spec.subnetId` | The OCID of the subnet the DB System is associated with.  | string | yes       |
| `spec.dataStorageSizeInGBs`| Initial size of the data volume in GBs that will be created and attached. Keep in mind that this only specifies the size of the database data volume, the log volume for the database will be scaled appropriately with its shape. | int    | yes       |
| `spec.isHighlyAvailable` | Specifies if the DB System is highly available.  | boolean | yes       |
| `spec.availabilityDomain`| The availability domain on which to deploy the Read/Write endpoint. This defines the preferred primary instance. | string | yes        |
| `spec.faultDomain`| The fault domain on which to deploy the Read/Write endpoint. This defines the preferred primary instance. | string | no        |
| `spec.configuration.id` | The OCID of the Configuration to be used for this DB System. [More info about Configurations](https://docs.oracle.com/en-us/iaas/mysql-database/doc/db-systems.html#GUID-E2A83218-9700-4A49-B55D-987867D81871)| string | yes |
| `spec.description` | User-provided data about the DB System. | string | no |
| `spec.hostnameLabel` | The hostname for the primary endpoint of the DB System. Used for DNS. | string | no |
| `spec.mysqlVersion` | The specific MySQL version identifier. | string | no |
| `spec.port` | The port for primary endpoint of the DB System to listen on. | int | no |
| `spec.portX` | The TCP network port on which X Plugin listens for connections. This is the X Plugin equivalent of port. | int | no |
| `spec.ipAddress` | The IP address the DB System is configured to listen on. A private IP address of your choice to assign to the primary endpoint of the DB System. Must be an available IP address within the subnet's CIDR. If you don't specify a value, Oracle automatically assigns a private IP address from the subnet. This should be a "dotted-quad" style IPv4 address. | string | no |
| `spec.freeformTags` | Free-form tags for this resource. Each tag is a simple key-value pair with no predefined name, type, or namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). `Example: {"Department": "Finance"}` | string | no |
| `spec.definedTags` | Defined tags for this resource. Each key is predefined and scoped to a namespace. For more information, see [Resource Tags](https://docs.oracle.com/iaas/Content/General/Concepts/resourcetags.htm). | string | no |
| `spec.adminUsername.secret.secretName` | The username for the administrative user. | string | yes       |
| `spec.adminPassword.secret.secretName` | The Kubernetes Secret Name that contains admin password for Mysql DbSystem. The password must be between 8 and 32 characters long, and must contain at least 1 numeric character, 1 lowercase character, 1 uppercase character, and 1 special (nonalphanumeric) character. | string | yes       |



## Mysql DbSystem Status Parameters

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

## Provisioning a Mysql DbSystem

Provisioning of a Mysql DbSystem requires the customer to input the admin username and admin password as a Kubernetes Secret, the OSOK acquires the admin usernmame and admin password from the kubernetes secret whose name is provided in the `spec`. 
The Kubernetes secret should contain the admin username in `username` field. 
The Kubernetes secret should contain the admin password in `password` field. 

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <ADMIN_SECRET_NAME>
type: Opaque
data:
  username: <USERNAME_BASE64_ENCODED>
  password: <PASSWORD_BASE64_ENCODED>
```

Run the following command to create a secret for the Mysql DbSystem:
```sh
kubectl apply -f <CREATE_SECRET>.yaml
```

The Mysql DbSystem can be accessed from the Secret which will be persisted as part of the provision/bind operation of the CR.

The OSOK Mysql DbSystem controller automatically provisions a Mysql DbSystem when customer provides mandatory fields to the `spec`. Following is a sample CR yaml for Mysql DbSystem.

- SUBNET_OCID - OCID of the subnet created in the pre-requisites step
- CONFIGURATION_ID - [More info about Configurations](https://docs.oracle.com/en-us/iaas/mysql-database/doc/db-systems.html#GUID-E2A83218-9700-4A49-B55D-987867D81871) Get your [Configuration_id](https://console.us-ashburn-1.oraclecloud.com/mysqlaas/configurations) 


```yaml

apiVersion: oci.oracle.com/v1beta1
kind: MySqlDbSystem
metadata:
  name: <CR_OBJECT_NAME>
spec:
  compartmentId: <COMPARTMENT_OCID>
  displayName: <DISPLAY_NAME>
  shapeName: <SHAPE>
  subnetId: <SUBNET_OCID>
  configuration:
    id: <CONFIGURATION_OCID>
  availabilityDomain: <AVAIALABILITY_DOMAIN>
  adminUsername:
    secret:
      secretName: <ADMIN_SECRET>
  adminPassword:
    secret:
      secretName: <ADMIN_SECRET>
  description: <DESCRIPTION>
  dataStorageSizeInGBs: <DB_SIZE>
  port: <PORT>
  portX: <PORTX>
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

Once the CR is created, OSOK will Reconcile and creates a Mysql DbSystem. OSOK will ensure the Mysql DbSystem instance is Available.

The Mysql DbSystem CR can list the DbSystems in the cluster as below: 
```sh
$ kubectl get mysqldbsystems
NAME                       STATUS         AGE
mysqldbsystems-sample      Active         4d
```

The Mysql DbSystem CR can list the DbSystems in the cluster with detailed information as below: 
```sh
$ kubectl get mysqldbsystems -o wide
NAME                         DISPLAYNAME     STATUS         OCID                                   AGE
mysqldbsystems-sample        BusyBoxDB       Active         ocid1.mysqldbsystem.oc1.iad.........   4d
```

The MysqlDbSystem CR can be described as below:
```sh
$ kubectl describe mysqldbsystems <NAME_OF_CR_OBJECT>
```

## Binding to an Existing Mysql DbSystem

OSOK allows customers to bind to an existing Mysql DbSystem. In this case, `Id` is the only required field in the CR `spec`.

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: MySqlDbSystem
metadata:
  name: <CR_OBJECT_NAME>
spec:
  id: <MYSQLDBSYSTEM_OCID>
```

Run the following command to create a CR that binds to an existing Mysql DbSystem:
```sh
kubectl apply -f <BIND_YAML>.yaml
```

## Updating a Mysql DbSystem

Customers can update the Mysql DbSystem. [Few parameters](https://docs.oracle.com/en-us/iaas/mysql-database/doc/managing-db-system.html#GUID-24D56090-C7E8-4A21-B450-BCBFAD231911) can be updated in this case.
```yaml
apiVersion: oci.oracle.com/v1beta1
kind: MySqlDbSystem
metadata:
  name: <CR_OBJECT_NAME>
spec:
  id: <MYSQLDBSYSTEM_OCID>
  displayName: <UPDATE_DISPLAY_NAME>
  description: <UPDATE_DESCRIPTION>
  configuration:
    id: <UPDATE_CONFIGURATION_ID>
  freeformTags:
    <KEY1>: <VALUE1>
  definedTags:
    <TAGNAMESPACE1>:
      <KEY1>: <VALUE1>
```

Run the following command to create a CR that updates an existing Mysql DbSystem:
```sh
kubectl apply -f <UPDATE_YAML>.yaml
```

## Access Information in Kubernetes Secrets

The Access information of a OCI Service or Resource will be created as a Kubernetes secret to manage the Mysql DbSystem. The name of the secret can be provided in the CR yaml or by default the name of the CR will be used.

Customer will get the access information as Kubernetes Secret to use the Mysql DbSystem. The following files/details will be made available to the user:

| Parameter           | Description                                                              | Type   |
| ------------------  | ------------------------------------------------------------------------ | ------ |
| `InternalFQDN`      | DNS endpoint                                                             | string |
| `MySQLPort`         | Mysql port                                                               | string |
| `MySQLXProtocolPort`| Mysql portx                                                              | string |
| `PrivateIPAddress`  | DbSystem's PrivateIPAddress                                              | string |
| `AvailabilityDomain`| AvailabilityDomain                                                       | string |
| `FaultDomain`       | FaultDomain                                                              | string |
| `Endpoints`         | Endpoints to connect to mysql db system                                  | json   |
 
