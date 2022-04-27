# Service Mesh

- [Introduction](#introduction)
- [Create Policies](#create-policies)
- [Service Mesh Resources](#service-mesh-resources)
- [Install Metrics Server](#install-metrics-server)


## Introduction

[Service Mesh](https://docs.oracle.com/iaas/Content/service-mesh/home.htm) is a product offering on Oracle Cloud Infrastructure (OCI) platform which provides logging, metrics, security, and traffic management features for micro-service deployments.
OCI Service Mesh allows customers to add a set of capabilities that enables microservices within a cloud native application to communicate with each other in a centrally managed and secure manner.

## Create Policies

### Set up a dynamic group for all cluster compute instances in your compartment.

```plain
Any {instance.id = 'ocid1.instance.oc1.iad..aaa...', instance.compartment.id = 'ocid1.compartment.oc1..aaa....'}
```

### Using your dynamic group, create the policies that give your compartment the required access for service mesh.

```plain
Allow dynamic-group <your-dynamic-group-name> to manage service-mesh-family in compartment <your-compartment-name>
Allow dynamic-group <your-dynamic-group-name> to use metrics in compartment <your-compartment-name>
Allow dynamic-group <your-dynamic-group-name> to use log-content in compartment <your-compartment-name>
Allow dynamic-group <your-dynamic-group-name> to use tag-namespaces in compartment <your-compartment-name>
Allow dynamic-group <your-dynamic-group-name> to manage leaf-certificates in compartment <your-compartment-name>
Allow dynamic-group <your-dynamic-group-name> to manage leaf-certificate-family in compartment <your-compartment-name>
Allow dynamic-group <your-dynamic-group-name> to use certificate-authority-delegate in compartment <your-compartment-name>
Allow dynamic-group <your-dynamic-group-name> to use certificate-authority-family in compartment <your-compartment-name>
Allow dynamic-group <your-dynamic-group-name> to inspect vault in compartment <your-compartment-name>
Allow dynamic-group <your-dynamic-group-name> to {MESH_PROXY_DETAILS_READ} in tenancy
Allow dynamic-group <your-dynamic-group-name> to manage certificate-associations in compartment <your-compartment-name>
Allow dynamic-group <your-dynamic-group-name> to manage certificate-authority-associations in compartment <your-compartment-name>
Allow any-user to manage keys in compartment <your-compartment-name> where any {request.principal.type='certificate', request.principal.type='certificateauthority'}
Allow any-user to manage object-family in compartment <your-compartment-name> where any {request.principal.type='certificate', request.principal.type='certificateauthority'}
```

## Service Mesh Resources
1. [Mesh](service-mesh/mesh.md)
2. [Virtual Service](service-mesh/virtual-service.md)
3. [Virtual Deployment](service-mesh/virtual-deployment.md)
4. [Virtual Service Route Table](service-mesh/virtual-service-route-table.md)
5. [Ingress Gateway](service-mesh/ingress-gateway.md)
6. [Ingress Gateway Route Table](service-mesh/ingress-gateway-route-table.md)
7. [Access Policy](service-mesh/access-policy.md)
8. [Virtual Deployment Binding](service-mesh/virtual-deployment-binding.md)
9. [Ingress Gateway Deployment](service-mesh/ingress-gateway-deployment.md)

##  Install Metrics Server

If an ingress gateway deployment is deployed, a metrics server managed by the deployment is required.
The Kubernetes Horizontal Pod Autoscalar uses the metrics server to scale the number of pods in the ingress gateway based on CPU usage.
The metrics server is installed using the following command:

```plain
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/high-availability.yaml
```
