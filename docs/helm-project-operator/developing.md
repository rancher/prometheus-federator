# Developing Helm Project Operator

## Repository Structure

```bash
## This directory contains a Helm chart that can be used to deploy Helm Project Operator in a Kubernetes cluster in the cattle-helm-system namespace,
## which deploys project-operator-example (located under charts/project-operator-example) on seeing a ProjectHelmChart with spec.helmApiVersion: dummy.cattle.io/v1alpha1.
charts/
  ## The main chart that deploys Helm Project Operator in the cluster.
  helm-project-operator/
  
  ## A dummy chart that is deployed onto the cluster on seeing a valid ProjectHelmChart (which means that it is contained within 
  ## a Project Registration Namespace with spec.helmApiVersion set to dummy.cattle.io/v1alpha1)
  ##
  ## This chart is not expected to ever be deployed standalone; it is embedded into the Helm Project Operator binary itself.
  project-operator-example/

## This directory will contain additional docs to assist users in getting started with using Helm Project Operator.
docs/

## This directory contains example ProjectHelmCharts that can be deployed that work on the default project-operator-example packaged with the Helm Project Operator
examples/

## This directory contains the image that is used to build rancher/helm-project-operator, which is hosted on hub.docker.com.
package/
  Dockerfile-helm-project-operator

## The main source directory for the code. See below for more details.
pkg/

## The Dockerfile-helm-project-operator used to run CI and other scripts executed by make in a Docker container (powered by https://github.com/rancher/dapper)
Dockerfile-helm-project-operator.dapper

## The file that contains the underlying actions that 'go generate' needs to execute on a call to it. Includes the logic for generating 
## controllers and updating the crds.yaml under the crds/ directory
generate.go

## The main entrypoint into Helm Project Operator; this serves as an example of how Helm Project Operator can be used.
## For a real-world example, please see the main.go on https://github.com/rancher/prometheus-federator.
dummy.go
```

## Making changes to the codebase (`pkg`)

Most of the code for Helm Locker is contained in the `pkg` directory, which has the following structure:

```bash
## This directory contains the definition of a ProjectHelmChart CR under project.go; if you need to add new fields to ProjectHelmChart CRs, this is
## where you would make the change
apis/

## These directories manage all the logic around 'go generate', including the creation of the 'generated/' directory that contains all the underlying
## controllers that are auto-generated based on the API definition of the ProjectHelmChart CR defined under 'apis/'
codegen/
crd/
version/
generated/

## This directory provides a utility function Init that allows projects implementing Helm Project Operator to quickly set up a Helm Project Operator
## instance based on provided options.
##
## For a real-world example of how this code is used, please see the main.go on https://github.com/rancher/prometheus-federator.
operator/

## These directories are the core controller directories that manage how the operator watches for Kubernetes resources
controllers/
  ## This directory is where all common code shared by all controllers is placed (e.g. options that can be provided, utility functions, constants, etc.)
  common/
  ## This directory is where logic for hardening Helm Project Operated namespaces exists
  hardened/
  ## This directory is where the logic for creating Project Registration Namespaces lives
  namespace/
  ## This directory is where the logic for creating Project Release Namespaces and underlying Helm releases via HelmChart and HelmRelease CRs on seeing
  ## changes to ProjectHelmCharts exist
  project/
  ## This is where the underlying context used by all controllers of this operator are registered, all using the same underlying SharedControllerFactory
  controller.go
  ## This is where the logic for parsing the values.yaml and questions.yaml from an embedded Helm chart (provided as a .tgz.base64 in ChartContent) exists
  parse.go
```

Within each of the directories under `pkg/controllers`, here are some important files:

```bash
## Where the core controller logic and all OnChange, OnRemove, or GeneratingHandlers live
controller.go
## Where all indexes that need to be registered for this controller live; indexers are added in order to allow for the operator to efficiently
## query the cache for the latest state of an object instead of requiring the operator to make list API calls to the Kubernetes API server any time
## it needs to know the state of dependent resources (e.g. HelmCharts, HelmReleases) on re-enqueing the parent resource (namespace, ProjectHelmChart)
indexers.go
## Where custom reconcilers live which allow for the operator to modify how wrangler.apply performs the upgrade of a resource. For example, the current
## usage of this code is in order to all a reconciler that deletes and recreates ConfigMaps instead of attempting to patch the resource.
reconcilers.go
## Where resolvers live, which are handlers that are triggered on dependent resources being modified that signal to the operator that the main parent
## resource should be re-enqueued. Generally, you need at least one resolver per resource created in resource.go to ensure that changes to the underlying
## resources are resynced on modification.
resolvers.go
## Where the definition of resources that are deployed on behalf of a parent resource lives.
resources.go
```

## Once you have made a change

If you modified `pkg/apis` or `generate.go`, make sure you run `go generate`.

Also, make sure you run `go mod tidy`.

## Creating a Docker image based off of your changes

To test your changes and create a Docker image to a specific Docker repository with a given tag, you should run `REPO=<my-docker-repo> TAG=<my-docker-tag> make` (e.g. `REPO=arvindiyengar TAG=dev make`), which will run the `./scripts/ci` script that builds, tests, validates, and packages your changes into a local Docker image (if you run `docker images`, it should show up as an image in the format `${REPO}/helm-project-operator:${TAG}`).

If you don't want to run all the steps in CI every time you make a change, you could also run the following one-liner to build and package the image:

```bash
REPO=<my-repo>
TAG=<my-tag>

./scripts/build-chart && GOOS=linux CGO_ENABLED=0 go build -ldflags "-extldflags -static -s" -o bin/helm-project-operator && REPO=${REPO} TAG=${TAG} make package
```

Once the image is successfully packaged, simply run `docker push ${REPO}/helm-project-operator:${TAG}` to push your image to your Docker repository.

## Testing a custom Docker image build

1. Ensure that your `KUBECONFIG` environment variable is pointing to your cluster (e.g. `export KUBECONFIG=<path-to-kubeconfig>; kubectl get nodes` should show the nodes of your cluster) and pull in this repository locally
2. Go to the root of your local copy of this repository and deploy the Helm Project Operator chart as a Helm 3 chart onto your cluster after overriding the image and tag values with your Docker repository and tag: run `helm upgrade --install --set image.repository="${REPO}/helm-project-operator" --set image.tag="${TAG}" --set image.pullPolicy=Always helm-project-operator -n cattle-helm-system charts/helm-project-operator`
> Note: Why do we set the Image Pull Policy to `Always`? If you update the Docker image on your fork, setting the Image Pull Policy to `Always` ensures that running `kubectl rollout restart -n cattle-helm-system deployment/helm-project-operator` is all you need to do to update your running deployment to the new image, since this would ensure redeploying a deployment triggers a image pull that uses your most up-to-date Docker image. Also, since the underlying Helm chart deployed by the operator (e.g. `project-operator-example`) is directly embedded into the Helm Project Operator image, you also do not need to update the Deployment object itself to see all the HelmCharts in your cluster automatically be updated to the latest embedded version of the chart.
3. Profit!