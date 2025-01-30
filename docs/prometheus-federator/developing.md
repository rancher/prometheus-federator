# Developing Prometheus Federator

The Prometheus Federator project is primarily composed of two components:
- The golang code that implements a [Helm Project Operator](../../cmd/helm-project-operator/README.md), and
- The charts used for `rancher-project-monitoring` which are ultimately based on `rancher-monitoring`
  - This component heavily relies on [`rancher/charts-build-scripts`](https://github.com/rancher/charts-build-scripts) and mimics a smaller `rancher/charts` for O&B team.

The golang half can be found here in this repo and the Charts half in our [rancher/ob-team-charts](https://github.com/rancher/ob-team-charts) repo.

In **most** circumstances, you will primarily make changes to the charts half and work in [rancher/ob-team-charts](https://github.com/rancher/ob-team-charts) repo first.
Then, after that bump versions here in `build.yaml` and create your RC/alpha release after that. Finally, you can land that change in `rancher/charts`.

When you want to make changes to Prometheus Federator directly you do not need to interact with [rancher/ob-team-charts](https://github.com/rancher/ob-team-charts) repo.

## Repository Structure

```bash
## This directory contains the source chart for the Prometheus Federator chart.
##
## At build time it is used as a template to generate a 1:1 release chart to the image tag.
## If you modify the Prometheus Federator chart you should consider if it needs to be back ported from `main` to other release branches.
charts/prometheus-federator

## This directory contains all CLI entry points of the Prometheus Federator (and internal projects)
cmd/
  ## The directory contains the helm-locker cli entrypoint used by devs.
  helm-locker
  ## The directory contains the helm-project-operator cli entrypoint used by devs.
  helm-project-operator
  ## The directory contains the prometheus-federator cli entrypoint
  prometheus-federator

## This directory will contain additional docs to assist users in getting started with using Prometheus Federator.
docs/

## This directory contains an example ProjectHelmChart that can be deployed to create an example Project Monitoring Stack
## Note: the namespace needs to be modified to be a valid Project Registration Namespace, depending on how you deployed the operator.
examples/

## This directory contains all the internal code used by Prometheus Federator
internal/
  ## The directory contains the helm-locker internal project code.
  helm-locker
  ## The directory contains the helm-project-operator internal project code.
  helm-project-operator

## This directory contains Dockerfile* to produce images for Prod and Dev
package/
  Dockerfile-helm-locker
  Dockerfile-helm-project-operator
  Dockerfile-prometheus-federator
```

---

> TODO: Migrate the following content to [rancher/ob-team-charts](https://github.com/rancher/ob-team-charts) repo.

## Making changes to the Helm Charts (`packages/`)

In most situations, the changes made to this repository will primarily be fixes to the Helm charts that either deploy the operator (`prometheus-federator`) or those that are deployed on behalf of the operator (`rancher-project-monitoring`, which embeds `rancher-project-grafana` within it as a subchart).

If you need to bump the version of Helm Project Operator embedded into the charts or binaries, generally you will need to bump the version of the Helm Project Operator in the `go.mod` and update the commit hash in `packages/prometheus-federator/generated-changes/dependencies/helmProjectOperator/dependency.yaml`; once done, run `go mod tidy` and make one commit with your changes entitled `Bump Helm Project Operator` followed by one commit with the output of running `unset PACKAGE; make charts` with the commit message `make charts`.

If you need to make changes to the Prometheus Federator chart itself, make the changes directly in the `packages/prometheus-federator/charts`; once done, make one or more commits that only contain your changes to the `packages/prometheus-federator/charts` directory with proper commit messages describing what you changed and make one commit at the end with the output of running `unset PACKAGE; make charts` with the commit message `make charts`.

If you need to make changes to the rancher-project-monitoring chart, follow the same steps above but start by running `PACKAGE=rancher-project-monitoring make prepare` to pull in the latest version of your `rancher-project-grafana` chart. Before you commit any changes, always make sure you run `PACKAGE=rancher-project-monitoring make clean` to avoid committing `packages/rancher-project-monitoring/charts/charts` (but be careful since `make clean` will wipe out any changes you made to that directory! It does the equivalent of `rm -rf packages/rancher-project-monitoring/charts/charts`).

If you need to make changes to the rancher-project-grafana chart, follow the same steps above but start by running `make prepare`, which will pull in the source Grafana chart referenced by the `packages/rancher-project-grafana/package.yaml`, apply the patches from `packages/rancher-project-grafana/generated-changes/*`, and render a `packages/rancher-project-grafana/charts` directory. From here, on every commit you make with changes to `packages/rancher-project-grafana/charts`, you will need to:
- Run `PACKAGE=rancher-project-grafana make patch` to generate changes that will be placed into `packages/rancher-project-grafana/generated-changes/*`. **Ensure that these changes show up in `packages/rancher-project-grafana/generated-changes/*` before you continue any further to avoid losing changes.**
- Run `PACKAGE=rancher-project-grafana make clean` to clean up your repository to get it ready for a commit. This will wipe out the `packages/rancher-project-grafana/charts` directory, so again make sure that these changes show up in `packages/rancher-project-grafana/generated-changes/*` before you run `make clean`.
- After committing, if you run `PACKAGE=rancher-project-grafana make prepare` again, you should see that your changes are persisted.
- Once you are ready with all of your changes, run `PACKAGE=rancher-project-monitoring make charts` to make the final commit with the commit message `make charts`, as done above.

> Note: since the `rancher-project-grafana` chart is only expected to be used as a subchart of the `rancher-project-monitoring` chart, a value on the `package.yaml` indicates `doNotRelease: true`; this is intentional and will prevent `PACKAGE=rancher-project-grafana make charts` from producing anything in the `charts/`, `assets/`, or `index.yaml`.

> Note: In general, it is recommended to use the experimental caching feature for rancher/charts-build-scripts to avoid multiple network calls to pull in the source repositories by storing them in a local cache under `.charts-build-scripts/.cache/*`. You can turn this on by default by setting `export USE_CACHE=1`.

For more information on how to make changes on repositories powered by `rancher/charts-build-scripts`, please read the [docs](https://github.com/rancher/charts-build-scripts/tree/master/templates/template/docs).

## Once you have made a change

If you modified `packages/`, make sure you run `unset PACKAGE; make charts` to generate the latest `charts/`, `assets/` and `index.yaml`.

Also, make sure you run `go mod tidy` if you make any changes to the code.

## Creating a Docker image based off of your changes

To test your changes and create a Docker image to a specific Docker repository with a given tag, you should run `REPO=<my-docker-repo> TAG=<my-docker-tag> make` (e.g. `REPO=arvindiyengar TAG=dev make`), which will run the `./scripts/ci` script that builds, tests, validates, and packages your changes into a local Docker image (if you run `docker images`, it should show up as an image in the format `${REPO}/prometheus-federator:${TAG}`).

If you don't want to run all the steps in CI every time you make a change, you could also run the following one-liner to build and package the image:

```bash
REPO=<my-repo>
TAG=<my-tag>

./scripts/build-chart && GOOS=linux CGO_ENABLED=0 go build -ldflags "-extldflags -static -s" -o build/bin/prometheus-federator && REPO=${REPO} TAG=${TAG} make package
```

Once the image is successfully packaged, simply run `docker push ${REPO}/prometheus-federator:${TAG}` to push your image to your Docker repository.

## Testing a custom Docker image build

1. Ensure that your `KUBECONFIG` environment variable is pointing to your cluster (e.g. `export KUBECONFIG=<path-to-kubeconfig>; kubectl get nodes` should show the nodes of your cluster) and pull in this repository locally
2. Go to the root of your local copy of this repository and deploy the Prometheus Federator chart as a Helm 3 chart onto your cluster after overriding the image and tag values with your Docker repository and tag: run `helm upgrade --install --set image.repository="${REPO}/prometheus-federator" --set image.tag="${TAG}" --set image.pullPolicy=Always prometheus-federator -n cattle-monitoring-system charts/prometheus-federator`
> Note: Why do we set the Image Pull Policy to `Always`? If you update the Docker image on your fork, setting the Image Pull Policy to `Always` ensures that running `kubectl rollout restart -n cattle-monitoring-system deployment/prometheus-federator` is all you need to do to update your running deployment to the new image, since this would ensure redeploying a deployment triggers a image pull that uses your most up-to-date Docker image. Also, since the underlying Helm chart deployed by the operator (e.g. `example-chart`) is directly embedded into the Helm Project Operator image, you also do not need to update the Deployment object itself to see all the HelmCharts in your cluster automatically be updated to the latest embedded version of the chart.
3. Profit!