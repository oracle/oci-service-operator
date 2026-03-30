# OCI Service Operator for Kubernetes

- [Introduction](#introduction)
- [Installation](installation.md#installation)
  - [Pre-Requisites](installation.md#pre-requisites)
  - [Install Operator SDK](installation.md#install-operator-sdk)
  - [Install Operator Lifecycle Manager (OLM)](installation.md#install-olm)
  - [Deploy OCI Service Operator for Kuberentes](installation.md#deploy-osok)
- [Services](services.md#services)
  - [Oracle Autonomous Database Service](adb.md#autonomous-databases-service)
    - [Introduction](adb.md#introduction)
    - [OCI Permission requirement](adb.md#oci-permission-requirement)
    - [Access Information in Kubernetes Secret](adb.md#access-information-in-kubernetes-secrets)
    - [Autonomous Database Specification Parameters](adb.md#autonomous-database-specification-parameters)
    - [Autonomous Database Status Parameters](adb.md#autonomous-database-status-parameters)
    - [Provision](adb.md#provisioning-an-adb)
    - [Bind](adb.md#binding-to-an-existing-adb)
    - [Update](adb.md#updating-an-adb)
  - [Oracle Streaming Service](oss.md#oracle-streaming-service)
    - [Introduction](oss.md#introduction)
    - [OCI Permission requirement](oss.md#oci-permission-requirement)
    - [Streaming Service Specification Parameters](oss.md#streams-service-specification-parameters)
    - [Streaming Service Status Parameters](oss.md#streams-service-status-parameters)
    - [Create Stream](oss.md#create-stream)
    - [Bind](oss.md#binding-to-an-existing-stream)
    - [Update](oss.md#updating-stream)
    - [Delete](oss.md#delete-stream)
  - [Oracle MySQL Database Service](mysql.md#oracle-mysql-database-service)
    - [Introduction](mysql.md#introduction)
    - [MySQL DbSystem Pre-requisites](mysql.md#pre-requisites-for-setting-up-mysql-dbsystems)
    - [MySQL DbSystem Specification Parameters](mysql.md#mysql-dbsystem-specification-parameters)
    - [MySQL DbSystem Status Parameters](mysql.md#mysql-dbsystem-status-parameters)
    - [Provision](mysql.md#provisioning-a-mysql-dbsystem)
    - [Update](mysql.md#updating-a-mysql-dbsystem)
    - [Kubernetes Secrets](mysql.md#kubernetes-secrets)

## Introduction

The OCI Service Operator for Kuberentes (OSOK) simplifies the process and provide a seamless experience for the container-native application in managing and connecting to OCI services/resources. 

OSOK is based on the [operator framework](https://operatorframework.io/) which is an open-source toolkit to manage Operators. It uses the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) library which provides high-level APIs and abstractions to write operational logic and also provides tools for scaffolding and code generation for operators.
