# Experimental: E2E CI Tests

## What does E2E CI do?

The E2E CI described in [.github/scripts/](../../../.github/workflows/hpo-e2e-ci.yaml) checks out the current Git repository, builds a Docker image using the repository's build scripts, sets up a [k3d](https://k3d.io) cluster, imports the built `helm-project-operator` image into the cluster (which automatically uses the latest `project-operator-example` chart since it is embedded into the binary as part of the build process), and then uses Helm to install `helm-project-operator` (using the Helm chart contained in the repository).

Once it is installed, it will run checks to ensure that all workloads are up and running in the Helm install and then mimic creating a Project (by creating a namespace with a particular label on it). 

On creating the Project, it asserts that the Registration Namespace is auto-created and installs the example ProjectHelmChart into that namespace, which triggers the deployment of the Example Chart in that namespace.

Finally, it deletes the ProjectHelmChart, asserts the helm uninstall Job on the Example Chart successfully completes, and then performs a Helm uninstall of the Helm Project Operator chart to ensure that it is not left hanging.

## Running the Github Workflow CI locally for testing

To run the end-to-end GitHub Workflow CI locally to test whether your changes work, it's recommended to install [`nektos/act`](https://github.com/nektos/act).

An slim image has been defined in [`.github/workflows/e2e/package/Dockerfile-helm-project-operator`](../../../.github/workflows/e2e/package/Dockerfile-helm-project-operator) that has the necessary dependencies to be used as a Runner for act for this GitHub Workflow. To build the image, run the following commmand (make sure you re-run it if you make any changes to add dependencies):

```bash
docker build -f ./.github/workflows/e2e/package/Dockerfile-helm-project-operator -t rancher/helm-project-operator-e2e:latest .
```

Once you have built the image and installed `act`, simply run the following command on the root of this repository and it will run your GitHub workflow within a Docker container:

```bash
act pull_request -j e2e-helm-project-operator -P ubuntu-latest=rancher/helm-project-operator-e2e:latest
```

> **Important Note**: When using local runs, `act` will create the k3d cluster locally in your system. It should automatically get deleted from your system at the end of a workflow run (failed or successful) at the end of CI, but if it does not execute make sure you clean it up manually via `k3d cluster delete e2e-ci-helm-project-operator`.

## Running E2E Tests on an already provisioned cluster

To verify that the functionality of Helm Project Operator on a live cluster that you have already configured your `KUBECONFIG` environment variable to point to, you can use the utility script found in [script/e2e-ci](../../../scripts/e2e-ci) to run the relevant CI commands to install Monitoring, install Helm Project Operator using your forked image, and run the remaining CI steps.

> **Note:** For now, this script only works on k3s, RKE1, and RKE2 clusters but it can be easily extended to work on different cluster types by supplying the right values in `install-helm-project-operator.sh` to enable and verify the correct cluster-type specific testing. Contributions are welcome!

However, to do this, your Helm Project Operator image will need to already be imported and accessible by the cluster you plan to run the scripts on, so make sure you push your image to a registry accessible by your cluster before running these scripts.

For example, if you wanted to run your tests on the `arvindiyengar/helm-project-operator:dev` image, you would run the following command:

```bash
KUBERNETES_DISTRIBUTION_TYPE=<rke|rke2|k3s> REPO=arvindiyengar TAG=dev ./scripts/e2e-ci
```

To enable debug logging, pass it in as an environment variable:

```bash
DEBUG=true KUBERNETES_DISTRIBUTION_TYPE=<rke|rke2|k3s> REPO=arvindiyengar TAG=dev ./scripts/e2e-ci
```

If you are pointing at a Rancher 2.6+ downstream cluster, the `KUBERNETES_DISTRIBUTION_TYPE` will be auto-inferred from the `cluster.management.cattle.io` named `local` that exists in every downstream cluster, so it can be omitted:

```bash
REPO=arvindiyengar TAG=dev ./scripts/e2e-ci
```

To skip uninstalling the Helm Project Operator chart (if you would like to perform some validations of your own after the fact), pass in `SKIP_UNINSTALL=true`:

```bash
SKIP_UNINSTALL=true REPO=arvindiyengar TAG=dev ./scripts/e2e-ci
```

To run it against the latest image, just run:

```bash
TAG=<LATEST_IMAGE_VERSION> ./scripts/e2e-ci
```
