# Making Charts Available on [`rancher/charts`](https://github.com/rancher/charts)

## On Any PR Merge

Any time a PR is merged into this repository, the chart should be mirrored to the corresponding package in all branches of [`rancher/charts`](https://github.com/rancher/charts) that represent active release lines (i.e. `dev-v2.6`, `dev-v2.7`). To do this, do the following steps:

Prior to making changes to [`rancher/charts`](https://github.com/rancher/charts), you will need to cut a GitHub tag / release to trigger CI into creating [the Project Operator Image on DockerHub](https://hub.docker.com/r/rancher/prometheus-federator):
1. Navigate to the page to [`Draft a new release`](https://github.com/rancher/prometheus-federator/releases/new)
2. On the `Choose a tag` dropdown, carefully type in the version **prefixed with `v`** that corresponds to the version of Prometheus Federator that was just merged in the PR (i.e. the value found on the `version` field of [`packages/prometheus-federator/charts/Chart.yaml`](../../packages/prometheus-federator/charts/Chart.yaml)).
3. Copy the tag name into the Release Name field (i.e. `vX.X.X`)
4. Click on the button that says `Generate release notes`
5. **Review all your changes**; once a tag is created, **it should never be deleted**.
6. Click on `Publish Release`

Once the release has been published, wait till the Drone build successfully finishes and ensure that [the Project Operator Repo on DockerHub](https://hub.docker.com/r/rancher/prometheus-federator) contains the newly built image.

Once this is done, do the following for each `dev-v2.X` branch that needs this change:
1. Fork [rancher/charts](https://github.com/rancher/charts) at `dev-v2.X`.
2. Open `packages/rancher-monitoring/rancher-project-monitoring/package.yaml`
  - Modify the `subdirectory` to have the right version of the chart in it (i.e. change the version).
  - Modify the `commit` to the latest merged commit hash.
  - Modify the `version` to have the right version of the chart in it (the same version you bumped in the subdirectory).
    - **Note:** This can be the same as the version in this [rancher/prometheus-federator](https://github.com/rancher/prometheus-federator); the version is irrelevant since this chart is anyways marked as `catalog.cattle.io/hidden: "true"`. The only reason why we release this chart onto `rancher/charts` is to ensure our airgap scripts are able to pick up and mirror these new images to the auto-generated `rancher-images.txt`.
3. Open `packages/rancher-monitoring/rancher-project-monitoring/generated-changes/dependencies/grafana/dependency.yaml` 
  - Modify the `subdirectory` to have the right version of the chart in it (i.e. change the version).
  - Modify the `commit` to the latest merged commit hash.
4. Open `packages/rancher-monitoring/prometheus-federator/package.yaml`
  - Modify the `subdirectory` to have the right version of the chart in it (i.e. change the version).
  - Modify the `commit` to the latest merged commit hash.
  - Modify the `version` **based on the [versioning rules outlined in the `rancher/charts` repository](https://github.com/rancher/charts#versioning-charts). This version generally will not be the same as the version in this repository since the `${Major}.${Minor}.${Patch}` of this version has a completely different meaning than what the version in this repository indicates.
5. Open `packages/rancher-monitoring/prometheus-federator/generated-changes/dependencies/helmProjectOperator/dependency.yaml`
  - Modify the `subdirectory` to have the right version of the chart in it (i.e. change the version).
  - Modify the `commit` to the latest merged commit hash.

Once all these files have been modified, follow the general guidelines on the repository (i.e. `make charts` and add to or modify the `release.yaml`) and make a PR to [`rancher/charts`](https://github.com/rancher/charts).

Once that PR has been merged, you are good to go!