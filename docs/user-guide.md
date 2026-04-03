# OSOK User Guide: Provision OCI MySQL with kro

> **Important:** Use this guide in a test or non-production environment first.
>
> This is the primary getting-started guide for OSOK users. It is a focused
> quickstart, not the full service catalog or the full OSOK install reference.

## What This Guide Shows

This guide demonstrates how to use:

- OSOK to provision an OCI MySQL DB System from Kubernetes
- kro to wrap the required Kubernetes and OSOK resources behind one user-facing
  API
- a single custom resource YAML to request the MySQL infrastructure

Once the `ResourceGraphDefinition` is installed, a user can request an OCI
MySQL DB System with one custom resource manifest.

## Scope

- This is a starter workflow for a single infrastructure example.
- It intentionally focuses on OCI MySQL infrastructure only.
- It does not replace the detailed references in [installation.md](installation.md),
  [mysql.md](mysql.md), or [services.md](services.md).

## Prerequisites

- A Kubernetes cluster in OCI, such as OKE, with `kubectl` access from your
  client machine.
- `helm` and `operator-sdk` installed locally.
- A test OCI environment where you can create Dynamic Groups, Policies, and
  MySQL DB Systems.
- A private subnet that the MySQL DB System can use.
- A MySQL configuration OCID that matches the MySQL shape and version you want
  to use.

## 1. Install OSOK for MySQL

This guide assumes the published MySQL package bundle, not the monolithic
operator bundle.

Follow [installation.md](installation.md#deploy-oci-service-operator-for-kubernetes)
and use the MySQL package bundle for this walkthrough:

```bash
operator-sdk run bundle ghcr.io/<REPOSITORY_OWNER>/oci-service-operator-mysql-bundle:v2.0.0-alpha
```

Use the authentication path from [installation.md](installation.md):

- Prefer [Instance Principal](installation.md#enable-instance-principal) when
  the cluster runs in OCI.
- Use [User Principal](installation.md#enable-user-principal) or
  [Security Token](installation.md#enable-security-token) only if that matches
  your environment.

For this guide, the MySQL controller runs in the
`oci-service-operator-mysql-system` namespace.

## 2. Install kro

Install kro with Helm:

```bash
helm install kro oci://registry.k8s.io/kro/charts/kro \
  --namespace kro-system \
  --create-namespace
```

Verify the installation:

```bash
kubectl get pods -n kro-system
kubectl get resourcegraphdefinitions.kro.run
```

For a pinned version or a raw-manifest install, use the official kro install
guide:
[Installing kro](https://kro.run/docs/getting-started/Installation).

This quickstart assumes kro's default test-oriented install path. If you run
kro with stricter aggregated RBAC, grant kro access to the resources used in
this guide: the generated `OsokMysqlSystem` instances, `Secrets`, and
`dbsystems.mysql.oracle.com`. See the official kro access-control guide:
[Access Control](https://kro.run/docs/advanced/access-control).

## 3. Configure OCI Permissions for MySQL

This guide assumes OSOK is running on OKE with Instance Principals.

Create a Dynamic Group that matches the Kubernetes worker nodes in the cluster
compartment:

```plain
Any {instance.compartment.id = '<KUBERNETES_CLUSTER_COMPARTMENT_OCID>'}
```

Then create policies like:

```plain
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to {SUBNET_READ, SUBNET_ATTACH, SUBNET_DETACH, VCN_READ, COMPARTMENT_INSPECT} in compartment id <COMPARTMENT_OCID>
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to manage mysql-family in compartment id <COMPARTMENT_OCID>
Allow dynamic-group <OSOK_DYNAMIC_GROUP> to use tag-namespaces in tenancy
```

If you need broader scope, adapt those statements for tenancy-wide permissions.
For more details, see [mysql.md](mysql.md#pre-requisites-for-setting-up-mysql-db-systems).

## 4. Gather the OCI Inputs

Before applying the kro definition, gather:

- `COMPARTMENT_OCID`: OCI compartment for the MySQL DB System
- `SUBNET_OCID`: private subnet OCID for the MySQL DB System
- `AVAILABILITY_DOMAIN`: OCI availability domain for the DB System
- `CONFIGURATION_OCID`: MySQL configuration OCID
- `MYSQL_SHAPE_NAME`: MySQL DB System shape supported in your tenancy

The checked-in OSOK MySQL sample uses `MySQL.2` as an example shape, but you
should replace that with a shape supported in your tenancy.

## 5. Apply the ResourceGraphDefinition

Save the following file as `osok-mysql-rgd.yaml` and apply it once as a
cluster-level platform definition:

```yaml
apiVersion: kro.run/v1alpha1
kind: ResourceGraphDefinition
metadata:
  name: osok-mysql-system
spec:
  schema:
    apiVersion: v1alpha1
    kind: OsokMysqlSystem
    spec:
      compartmentId: string | required=true
      subnetId: string | required=true
      availabilityDomain: string | required=true
      configurationId: string | required=true
      mysqlShapeName: string | required=true
      mysqlAdminUsername: string | default="dbadmin"
      mysqlAdminPassword: string | required=true
      mysqlStorageSizeInGBs: integer | default=50
    status:
      dbSystemName: ${mysqlDbSystem.metadata.name}
      dbSystemOCID: ${mysqlDbSystem.status.?status.?ocid.orValue("")}
      connectionSecretName: ${mysqlDbSystem.metadata.name}
  resources:
    - id: mysqlAdminSecret
      template:
        apiVersion: v1
        kind: Secret
        metadata:
          name: ${schema.metadata.name + "-mysql-admin"}
          namespace: ${schema.metadata.namespace}
        type: Opaque
        stringData:
          username: ${schema.spec.mysqlAdminUsername}
          password: ${schema.spec.mysqlAdminPassword}

    - id: mysqlDbSystem
      readyWhen:
        - ${mysqlDbSystem.status.?lifecycleState.orValue("") == "ACTIVE"}
      template:
        apiVersion: mysql.oracle.com/v1beta1
        kind: DbSystem
        metadata:
          name: ${schema.metadata.name}
          namespace: ${schema.metadata.namespace}
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}
          shapeName: ${schema.spec.mysqlShapeName}
          subnetId: ${schema.spec.subnetId}
          configurationId: ${schema.spec.configurationId}
          availabilityDomain: ${schema.spec.availabilityDomain}
          adminUsername:
            secret:
              secretName: ${mysqlAdminSecret.metadata.name}
          adminPassword:
            secret:
              secretName: ${mysqlAdminSecret.metadata.name}
          dataStorageSizeInGBs: ${schema.spec.mysqlStorageSizeInGBs}
          port: 3306
          portX: 33060
```

Apply it:

```bash
kubectl apply -f osok-mysql-rgd.yaml
kubectl get rgd osok-mysql-system
```

When the `ResourceGraphDefinition` is active, kro has created a new
`OsokMysqlSystem` API in the cluster.

## 6. Create the MySQL Infrastructure with One YAML

Save the following as `osok-mysql-system.yaml`:

```yaml
apiVersion: kro.run/v1alpha1
kind: OsokMysqlSystem
metadata:
  name: mysql-demo
  namespace: demo
spec:
  compartmentId: ocid1.compartment.oc1..exampleuniqueID
  subnetId: ocid1.subnet.oc1..exampleuniqueID
  availabilityDomain: <AVAILABILITY_DOMAIN>
  configurationId: ocid1.mysqlconfiguration.oc1..exampleuniqueID
  mysqlShapeName: MySQL.2
  mysqlAdminUsername: dbadmin
  mysqlAdminPassword: change-me-for-test-only
  mysqlStorageSizeInGBs: 50
```

Create the namespace, then apply the single instance manifest:

```bash
kubectl create namespace demo
kubectl apply -f osok-mysql-system.yaml
```

For this quickstart, the MySQL admin password is supplied on the custom
resource for simplicity. For a production-grade design, move the credentials to
a pre-created Secret and model that Secret as an external dependency instead of
storing the password in the instance spec.

## 7. Verify Progress

MySQL provisioning takes the longest. Watch the stack with:

```bash
kubectl get OsokMysqlSystem -n demo
kubectl get dbsystems.mysql.oracle.com -n demo
kubectl get secrets -n demo
```

Inspect the `DbSystem` resource:

```bash
kubectl describe dbsystem mysql-demo -n demo
```

OSOK creates a same-name Secret after the `DbSystem` becomes active. Inspect
that connection Secret with:

```bash
kubectl get secret mysql-demo -n demo
kubectl get secret mysql-demo -n demo -o jsonpath='{.data.PrivateIPAddress}' | base64 --decode && echo
```

That Secret is the handoff point for any workload that needs to connect to the
MySQL DB System later.

## 8. Release-Aligned Notes for This Guide

This guide reflects the current OSOK surface in this branch:

- The MySQL custom resource is `apiVersion: mysql.oracle.com/v1beta1` with
  `kind: DbSystem`.
- The MySQL spec field is `configurationId`, not the older nested
  `configuration.Id` shape.
- Admin credentials are passed through
  `spec.adminUsername.secret.secretName` and
  `spec.adminPassword.secret.secretName`.
- After the `DbSystem` becomes active, OSOK creates a same-name Secret with
  connection data such as `PrivateIPAddress`, `MySQLPort`,
  `MySQLXProtocolPort`, and related endpoint metadata.

## 9. Basic Cleanup

Delete the instance first so kro and OSOK can tear down the managed resources:

```bash
kubectl delete OsokMysqlSystem mysql-demo -n demo
```

If you also want to remove the generated API:

```bash
kubectl delete rgd osok-mysql-system
```

If you installed kro only for this demo:

```bash
helm uninstall kro -n kro-system
```
