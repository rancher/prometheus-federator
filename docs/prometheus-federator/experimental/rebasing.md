# Experimental: Rebasing rancher-project-monitoring via scripts

### Context

The source of the [`rancher-project-monitoring` chart](../../charts/rancher-project-monitoring/) is the [`rancher-monitoring` chart in the `rancher/charts` repository under the `dev-v2.7` branch](https://github.com/rancher/charts/tree/dev-v2.7/charts/rancher-monitoring), which itself is sourced from the Helm community's [`kube-prometheus-stack` chart](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack).

> **Note:** Why do we pull the source from `rancher-monitoring` in this convoluted process?
>
> While we could pull from `kube-prometheus-stack` directly, a lot more maintenence burden would be required to maintain the patches that we add onto the `rancher-monitoring` chart to get the upstream chart ready for deployment in a Rancher setup (such as adding nginx reverse proxy pods in front of Prometheus and Grafana or custom Rancher dashboards).
>
> The `rancher-monitoring` chart is also heavily tested in every release by Rancher's QA team within various Rancher setups.

Therefore, in order to rebase the `rancher-project-monitoring` chart against the latest `rancher-monitoring` chart that will be released, you typically will need to run the following command:

```bash
PACKAGE=rancher-project-monitoring TO_COMMIT=<commit-hash-in-rancher-charts> TO_DIR=charts/rancher-monitoring/<version> make rebase
```

On running this command locally, the script will automatically pull in the `rancher/charts` repository as a Git remote, construct the patch from the current chart base (listed in the [`package.yaml` of `rancher-project-monitoring`](../../../packages/rancher-project-monitoring/package.yaml)) to the new chart base (defined from the environment variables provided, namely `TO_REMOTE`, `TO_COMMIT` , `TO_DIR`), and try to `git apply -3` the patches onto the current version of the charts created by running the `make prepare` command.

On applying the 3-way merge from the `git apply` command, the script will automatically create a shell (titled `interactive-rebase-shell`) that allows you to look through the changes that have been absorbed from upstream, resolve any conflicts (using the same Git conflict resolution experience you would have on executing a `git rebase -i`), and add all your changes to `staging` (`git add` **only**; the script will force you to stage any unstaged or committed changes if you try to).

Once all your conflicts have been resolved (which you can check by running `git diff --check` **before** exiting the `interactive-rebase-shell`), you can simply run `exit` and the script will take care of updating everything else for you by running `make patch` on the new base to produce two new commits.

### Rebasing the Grafana Dependency

Once you have made the changes using the rebase script, you will need to manually rebase the Grafana dependency as well. To do this, take the following steps:

1. Run `export PACKAGE=rancher-project-monitoring; make prepare` to prepare the working directory using your existing base
2. Modify the `.url`, `.subdirectory`, and `.commit` fields in [packages/rancher-project-monitoring/generated-changes/dependencies/grafana/dependency.yaml](../../../packages/rancher-project-monitoring/generated-changes/dependencies/grafana/dependency.yaml) to reflect the contents of [packages/rancher-project-monitoring/package.yaml](../../../packages/rancher-project-monitoring/package.yaml). **Note: The subdirectory should be the same as `rancher-project-monitoring` with `charts/grafana` appended after.**
3. Run `make patch`. This should regenerate [packages/rancher-project-monitoring/generated-changes](../../../packages/rancher-project-monitoring/generated-changes).
4. Add all your changes and create a commit, i.e. `Update grafana dependency to new base ${TO_COMMIT}`

### Once you have successfully run the scripts and rebased Grafana

1. Bump the minor version listed under [`packages/prometheus-federator/charts/Chart.yaml`](../../../packages/prometheus-federator/charts/Chart.yaml) under `appVersion` and `version` and reset the patch version (i.e. `0.1.1` -> `0.2.0`); they should both match.
1. Update the tag in [`packages/prometheus-federator/charts/values.yaml`](../../../packages/prometheus-federator/charts/values.yaml) under `helmProjectOperator.image.tag` to `v<VERSION>`, where `<VERSION>` is the version you identified in the previous step (i.e. `0.2.0`)
1. Modify the `version` field under [`packages/rancher-project-monitoring/package.yaml`](../../../packages/rancher-project-monitoring/package.yaml) to the same version from above (i.e. `0.2.0`)
1. Modify the `VERSION` environment variable under [`scripts/build-chart`](../../../scripts/build-chart) to the same version (i.e. `0.2.0`)
1. Run `make charts`; this should produce:
  - `assets/prometheus-federator/prometheus-federator-<VERSION>.tgz`
  - `assets/rancher-project-monitoring/rancher-project-monitoring-<VERSION>.tgz`
  - `charts/prometheus-federator/<VERSION>/*`
  - `charts/rancher-project-monitoring/<VERSION>/*`
  - `index.yaml` (modified)

### Validating your chart and making changes before filing a PR

Once you have created the new charts and assets, you should be ready to test your chart out locally to validate its functionality. To do so, you should take the following steps **at minimum** to ensure tht the functionality is not broken:

1. Make sure that a basic `helm template` command on the underlying Helm chart that will be deployed passes: `VERSION=<VERSION> helm template rancher-project-monitoring -n cattle-monitoring-system charts/prometheus-federator/${VERSION}`
2. Make sure GitHub Workflow CI passes.
3. Spin up an RKE1 and RKE2 cluster and ensure that any tests that run on GitHub Workflow CI pass on those clusters.
