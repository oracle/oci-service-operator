# Troubleshooting guide for OCI Service Operator for Kubernetes (OSOK)

## Operator Lifecycle Manager (OLM) Installation Issues

### OLM Installation Status
In general verify the status of OLM installation in the cluster using the below command:

```bash
$ operator-sdk olm status
```

Expected output of above command : 
```bash
INFO[0016] Fetching CRDs for version "0.18.2"
INFO[0016] Fetching resources for resolved version "v0.18.2"
INFO[0061] Successfully got OLM status for version "0.18.2"

NAME                                            NAMESPACE    KIND                        STATUS
operators.operators.coreos.com                               CustomResourceDefinition    Installed
operatorgroups.operators.coreos.com                          CustomResourceDefinition    Installed
operatorconditions.operators.coreos.com                      CustomResourceDefinition    Installed
installplans.operators.coreos.com                            CustomResourceDefinition    Installed
clusterserviceversions.operators.coreos.com                  CustomResourceDefinition    Installed
olm-operator                                    olm          Deployment                  Installed
olm-operator-binding-olm                                     ClusterRoleBinding          Installed
operatorhubio-catalog                           olm          CatalogSource               Installed
olm-operators                                   olm          OperatorGroup               Installed
aggregate-olm-view                                           ClusterRole                 Installed
catalog-operator                                olm          Deployment                  Installed
subscriptions.operators.coreos.com                           CustomResourceDefinition    Installed
aggregate-olm-edit                                           ClusterRole                 Installed
olm                                                          Namespace                   Installed
global-operators                                operators    OperatorGroup               Installed
operators                                                    Namespace                   Installed
packageserver                                   olm          ClusterServiceVersion       Installed
olm-operator-serviceaccount                     olm          ServiceAccount              Installed
catalogsources.operators.coreos.com                          CustomResourceDefinition    Installed
system:controller:operator-lifecycle-manager                 ClusterRole                 Installed
```
If the output of the OLM installation is having any failures, please uninstall and re-install the OLM into the cluster.

```bash
## Uninstall the OLM
$ operator-sdk olm uninstall

## Install the OLM
$ operator-sdk olm install
```

### OLM Installation Issues

#### OLM installation fails with below error 
```bash
FATA[0055] Failed to install OLM version "latest": detected existing OLM resources: OLM must be completely uninstalled before installation
```

Cleanup and Re-Install the OLM as below:
```bash
$ operator-sdk olm uninstall
```

If the above command fails, identify which version of OLM from below command

```bash
$ operator-sdk olm status
```

**Option 1 :**
Run the below command to uninstall OLM using version
```bash
$ operator-sdk olm uninstall --version <OLM_VERSION>
```

**Option 2 :**
Run the below command to uninstall OLM and its related components
```bash
$ kubectl -n olm get csvs
$ export OLM_RELEASE=<OLM_VERSION>
$ kubectl delete apiservices.apiregistration.k8s.io v1.packages.operators.coreos.com
$ kubectl delete -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/${OLM_RELEASE}/crds.yaml
$ kubectl delete -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/${OLM_RELEASE}/olm.yaml
```
**Option 3 :**
In case OLM uninstall still fails, run below commands to uninstall OLM and its related components 
```bash
$ kubectl delete apiservices.apiregistration.k8s.io v1.packages.operators.coreos.com
$ kubectl delete -f https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/master/deploy/upstream/quickstart/crds.yaml
$ kubectl delete -f https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/master/deploy/upstream/quickstart/olm.yaml
```

#### Verify OLM has been uninstalled successfully 
```bash
$ kubectl get namespace olm
Error from server (NotFound): namespaces olm not found
```
Verify that OLM has been uninstalled successfully by making sure that OLM owned CustomResourceDefinitions are removed:
```bash
$ kubectl get crd | grep operators.coreos.com
```


### OSOK Deployment Issues

If OSOK installation using OLM fails with below error message
```bash
FATA[0125] Failed to run bundle upgrade: error waiting for CSV to install: timed out waiting for the condition
```
The error signifies that during the installation of the OSOK, it timed out waiting for the condition. To mitigate this issue. Try to delete the bundle pod of the OSOK version we are trying to deploy
```bash
$ kubectl get pods | grep oci-service-operator-bundle

$ kubectl delete pod <POD_FROM_ABOVE_COMMAND>
```
After the bundle pod is deleted, re-install the OSOK bundle
```bash
$ operator-sdk run bundle iad.ocir.io/oracle/oci-service-operator-bundle:<VERSION>
## or for Upgrade
$ operator-sdk run bundle-upgrade iad.ocir.io/oracle/oci-service-operator-bundle:<VERSION>
```

Verify the OSOK is deployed successfully 
```bash
$ kubectl get deployments -n $NAMESPACE | grep "oci-service-operator-controller-manager"
..

NAME                                      READY   UP-TO-DATE   AVAILABLE   AGE
oci-service-operator-controller-manager   1/1     1            1           2d20h
```

If all the replicas in deployment is not running, verify deployment logs for specific issue using below commands : 
```bash
$ kubectl logs deploy/oci-service-operator-controller-manager -n $NAMESPACE -f
```


### OSOK Pods Issues
Verify the OSOK pods are running successfully
```bash
$ kubectl get pods -n $NAMESPACE | grep "oci-service-operator-controller-manager"

oci-service-operator-controller-manager-5fcf985fd7-zj7d9          1/1     Running     0          2d22h
```

If the pods not running, verify pod logs for specific issue using below commands :
```bash
$ kubectl logs pod/oci-service-operator-controller-manager-5fcf985fd7-zj7d9 -n $NAMESPACE -f 
```

Note : By default, operator-sdk installs OSOK bundle in 'default' namespace, however  we can specify the namespace while installing OSOK using '-n $NAMESPACE'. The same namespace has to be used while querying for the associated pods/deployments in above command samples to get the resources.

## Debugging Custom Resource (CR) Issues
If CR creation fails, monitor the OSOK controller pod logs (with steps outlined above) to understand the corresponding error code. Below are few of the commonly encountered failure scenarios : 

1. **Authorization failed or requested resource not found**
 ```bash
"message": Failed to create or update resource: Service error:NotAuthorizedOrNotFound. Authorization failed or requested resource not found.. http status code: 404.
```
This happens mostly due to user authorization. Follow below steps for remediation : 
* Check if the instance principals are configured correctly for the OCI resource being provisioned. 
* If using User credentials, cross verify if the secret for user credentials (ocicredentials as per installation doc) is populated correctly.
* Note that OSOK uses user credentials for authorization if the secret 'ocicredentials' is available during installation. Else it uses instance principal by default. Delete the secret 'ocicredentials' if user principals are not intended and restart the deployment to switch to Instance principals

2. **Secret \"admin-password\" not found**
```bash
ERROR	service-manager.AutonomousDatabases	Error while getting the admin password secret	{"error": "Secret \"admin-password\" not found"}
``` 
Ensure that secret admin-password (name as specified in the yaml) is present in the current namespace of CR. Sample secret : 
```bash
kubectl create secret generic admin-password --from-literal=password=Sample@1234
```

3. **Password key in admin/wallet password secret not found**
```bash
ERROR	service-manager.AutonomousDatabases	password key in admin password secret is not found
```
 Ensure that secret admin-password/wallet-password (name as specified in the yaml) is present in the current namespace of CR and has a field by key-name as password. Sample secret : 
 ```bash
kubectl create secret generic admin-password --from-literal=password=Sample@1234

kubectl create secret generic wallet-password --from-literal=walletpassword=Sample@1234
```

4. **Service error:InvalidParameter**
```
Sample error msg : 
ERROR	service-manager.AutonomousDatabases	Create AutonomousDatabase failed	{"error": "Service error:InvalidParameter. The Autonomous Database name cannot be longer than 14 characters
```
Invalid parameter error happens when one of the parameter for the associated OCI resource is not valid or not as per the specification. Check the specifications for the parameter being reported as invalid from the documentation page of the associated resource and update the same in the yaml for the CR. Parameter specifications : 
* AutonomousDB : https://docs.oracle.com/en-us/iaas/api/#/en/database/20160918/AutonomousDatabase/
* MySql : https://docs.oracle.com/en-us/iaas/api/#/en/mysql/20190415/DbSystem/
* Streaming : https://docs.oracle.com/en-us/iaas/api/#/en/streaming/20180418/Stream/

5. **If the CR creation fails with any 5XX error :**
* Contact respective service team from Oracle for support with details of the request (opc-id) and failure message

