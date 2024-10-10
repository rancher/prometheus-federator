# Developing Helm Locker

## Repository Structure

```bash
## This directory contains Helm charts that can be used to deploy Helm Locker in a Kubernetes cluster in the cattle-helm-system namespace
charts/

  ## The main chart that deploys Helm Locker in the cluster.
  helm-locker/
  
  ## A dummy chart that can be deployed as a Helm release in the cluster under the release name 'helm-locker-example' and the namespace 'cattle-helm-system'
  ##
  ## By default, it deploys with a HelmRelease CR that targets itself.
  ##
  ## Depends on 'helm-locker' being deployed onto the cluster first.
  helm-locker-example/

## This directory will contain additional docs to assist users in getting started with using Helm Locker
docs/

## This directory contains the image that is used to build rancher/helm-locker, which is hosted on hub.docker.com
package/
  Dockerfile-helm-project-operator

## The main source directory for the code. See below for more details.
pkg/

## The Dockerfile-helm-project-operator used to run CI and other scripts executed by make in a Docker container (powered by https://github.com/rancher/dapper)
Dockerfile-helm-project-operator.dapper

## The file that contains the underlying actions that 'go generate' needs to execute on a call to it. Includes the logic for generating controllers and updating crds.yaml under the crds/ directory
generate.go

## The main entrypoint into HelmLocker
main.go
```

## Making changes to the codebase (`pkg`)

Most of the code for Helm Locker is contained in the `pkg` directory, which has the following structure:

```bash
## This directory contains the definition of a HelmRelease CR under release.go; if you need to add new fields to HelmRelease CRs, this is where you would make the change
apis/

## These directories manage all the logic around 'go generate', including the creation of the 'generated/' directory that contains all the underlying controllers that are auto-generated based on the API definition of the HelmRelease CR defined under 'apis/'
codegen/
crd/
version/
generated/

## These directories are the core controller directories that manage how the operator watches HelmReleases and executes operations on the underlying in-memory ObjectSet LockableRegister (Lock, Unlock, Set, Delete)
controllers/
  ## This directory is where logic is defined for watching Helm Release Secrets targeted by HelmReleases and automatically keeping resources locked or unlocked
  release/
  ## This is where the underlying context used by all controllers of this operator are registered, all using the same underlying SharedControllerFactory
  controller.go
## A utility package to help wrap getting Helm releases via Helm library calls
releases/

## These directories implement an object that satisfies the LockableRegister interface; it is used as an underlying set of libraries that Helm Locker calls upon to achieve locking or unlocking HelmReleases (tracked as ObjectSets, or a []runtime.Object) and dynamically starting controllers based on GVKs observed in tracked object sets
gvk/
informerfactory/
objectset/
```

## Once you have made a change

If you modified `pkg/apis` or `generate.go`, make sure you run `go generate`.

Also, make sure you run `go mod tidy`.

## Creating a Docker image based off of your changes

To test your changes and create a Docker image to a specific Docker repository with a given tag, you should run `REPO=<my-docker-repo> TAG=<my-docker-tag> make` (e.g. `REPO=arvindiyengar TAG=dev make`), which will run the `./scripts/ci` script that builds, tests, validates, and packages your changes into a local Docker image (if you run `docker images`, it should show up as an image in the format `${REPO}/helm-locker:${TAG}`).

If you don't want to run all the steps in CI every time you make a change, you could also run the following one-liner to build and package the image:

```bash
REPO=<my-repo>
TAG=<my-tag>

GOOS=linux CGO_ENABLED=0 go build -ldflags "-extldflags -static -s" -o bin/helm-locker && REPO=${REPO} TAG=${TAG} make package
```

Once the image is successfully packaged, simply run `docker push ${REPO}/helm-locker:${TAG}` to push your image to your Docker repository.

## Testing a custom Docker image build

1. Ensure that your `KUBECONFIG` environment variable is pointing to your cluster (e.g. `export KUBECONFIG=<path-to-kubeconfig>; kubectl get nodes` should show the nodes of your cluster) and pull in this repository locally
2. Go to the root of your local copy of this repository and deploy the Helm Locker chart as a Helm 3 chart onto your cluster after overriding the image and tag values with your Docker repository and tag: run `helm upgrade --install --set image.repository="${REPO}/helm-locker" --set image.tag="${TAG}" --set image.pullPolicy=Always helm-locker -n cattle-helm-system charts/helm-locker`
> Note: Why do we set the Image Pull Policy to `Always`? If you update the Docker image on your fork, setting the Image Pull Policy to `Always` ensures that running `kubectl rollout restart -n cattle-helm-system deployment/helm-locker` is all you need to do to update your running deployment to the new image, since this would ensure redeploying a deployment triggers a image pull that uses your most up-to-date Docker image.
3. Profit!