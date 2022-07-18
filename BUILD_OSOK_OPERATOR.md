# OCI Service Operator for Kubernetes

## OSOK Operator Build

1. Build the source code of OSOK Operator
    ```
    make build
    ```

2. Build the docker image of the OSOK container and push it to Docker Repository.<br/>
   The below command requires few arguments such as,<br/>
    * IMG = Docker Repository URL where the image will be pushed
    * VERSION = Version of Operator image
        ```
        make docker-build docker-push IMG=DOCKER_REPOSITORY_IMG_URL VERSION=X.X.X
        ```

    Below is an example for pushing it to Oracle Cloud Infrastructure Registry for Version 1.0.0.<br/>
    ```
    make docker-build docker-push IMG=<region-key>.ocir.io/<tenancy-namespace>/<repo-name>:<tag> VERSION=1.0.0
    ```

3. Build the docker image of OSOK Bundle and push it to Docker Repository.<br/>
   The below command required few arguments such as,<br/>
    * BUNDLE_IMG = Docker Repository URL where the image will be pushed.<br/>
      BUNDLE_IMG URL should match the exact IMG URL mentioned in Step 2 and also it should be suffixed with `bundle` keyword.
        ```
        make bundle-build bundle-push BUNDLE_IMG=DOCKER_REPOSITORY_IMG_URL
        ```

   Below is an example for pushing it to Oracle Cloud Infrastructure Registry.<br/>
   ```
   make bundle-build bundle-push BUNDLE_IMG=<region-key>.ocir.io/<tenancy-namespace>/<repo-name>-bundle:<tag>
   ```