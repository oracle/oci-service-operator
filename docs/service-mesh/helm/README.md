# Managing Service Mesh with Helm

Service Mesh supports Helm Kubernetes management tool.
Using Helm Charts, you define, install, and upgrade Kubernetes application.
For information on installing Helm, see [Installing Helm](https://helm.sh/docs/intro/install/).

Using Helm with Service Mesh

The following high-level steps describe how to use Helm with Service Mesh and Kubernetes.
1. Create a Helm chart.
The command creates a sample chart with the folder structure shown here: [Helm Create](https://helm.sh/docs/helm/helm_create/).
   
   ```helm create <CHART_NAME>```
2. Change into the generated folder.
3. Modify the `templates` folder to include the Service Mesh resources in the yaml files as documented at [mesh-templates](./templates/mesh-templates)
4. Update the `values.yaml` file with Service Mesh variables. 
5. Generate the template files for preview.
   
   ```helm template .```
6. Install the Helm chart on the Kubernetes cluster.

   ```helm install <CHART_NAME> .```
