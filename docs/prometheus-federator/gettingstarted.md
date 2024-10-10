# Getting Started

## Simple Installation

### Prerequisites

In order to install Prometheus Federator, you first need to have Prometheus Operator deployed; it is also suggested that you configure at least one Cluster Prometheus that is collecting metrics using common exporters (kube-state-metrics, node-exporter, etc.) for the dashboards to work out-of-the-box.

It is recommended that you install either:
- [`rancher-monitoring`](https://rancher.com/docs/rancher/v2.6/en/monitoring-alerting/), which should work out-of-the-box with Prometheus Federator (see the [`README.md` on the Helm Chart](../../packages/prometheus-federator/charts/README.md) for more information on how to optimally configure rancher-monitoring to work with Prometheus Federator)
- [`kube-prometheus-stack`](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack), which should work with minor modifications to Prometheus Federator.
> Note: this is currently untested but we welcome contributions to update our docs to help users get supported for using Prometheus Federator with vanilla kube-prometheus-stack! In theory, the only changes that should be necessary would be to configure the Prometheus Federator chart to use a different `helmProjectOperator.valuesOverrides.federate.targets` and to override all the selectors on the Prometheus Operator resources (or Grafana Sidecars) in the underlying chart; it could also be possible that this would work out-of-the-box with `kube-prometheus-stack` as long as you supply `.Values.nameOverride=rancher-monitoring` and `.Values.namespaceOverride=cattle-monitoring-system`.

Once installed, you can proceed with the next steps.

### In Rancher (via Apps & Marketplace)

1. Navigate to `Apps & Marketplace -> Repositories` in your target downstream cluster and create a Repository that points to a `Git repository containing Helm chart or cluster template definitions` where the `Git Repo URL` is `https://github.com/rancher/prometheus-federator` and the `Git Branch` is `main`
2. Navigate to `Apps & Marketplace -> Charts`; you should see a chart under the new Repository you created: `Prometheus Federator`. 
3. Install `Prometheus Federator`

### In a normal Kubernetes cluster (via running Helm 3 locally)

Install `prometheus-federator` onto your cluster via Helm to install Prometheus Federator:

```
helm install -n cattle-monitoring-system prometheus-federator charts/prometheus-federator
```

### Checking if ProjectHelmCharts work

1. Ensure that the logs of `prometheus-federator` in the `cattle-monitoring-system` namespace show that the controller was able to acquire a lock and has started in that namespace
2. Deploy a ProjectHelmChart into a Project Registration Namespace (see [docs/design.md](docs/design.md) for more information on how to identify this)
3. Check to see if a HelmChart CR was created on behalf of that ProjectHelmChart in the Operator / System (`cattle-monitoring-system`) namespace
4. Find the Job in the Operator / System (`cattle-monitoring-system`) namespace tied to the HelmChart object to view the Helm operation logs that were performed on behalf of the HelmChart resource created; these logs should show as successful.
5. Check to see if a HelmRelease CR was created on behalf of that ProjectHelmChart in the Operator / System (`cattle-monitoring-system`) namespace
6. Ensure that the status of the HelmRelease CR shows that it has successfully found the Helm release secret for the Helm chart deployed by the HelmChart CR.
7. Locate the Project Release Namespace (see [docs/design.md](docs/design.md) for more information on how to identify this) and ensure that a Project Monitoring Stack was deployed onto that namespace
8. Try to modify or delete the resources that comprise the Project Monitoring stack; you should see that they are instantly recreated or fixed back into place.
9. Try supplying overrides to the deployed Helm chart (e.g. set `alertmanager.enabled` to false); on supplying new YAML to the ProjectHelmChart, you should see the Helm Operator Job (deployed on behalf of the HelmChart resource) be modified and you should observe that the HelmRelease CR emits an event (observable by running `kubectl describe -n cattle-monitoring-system <helm-release>` on the HelmRelease object) that indicates that it is Transitioning and then Locked; the release number will also be updated.
10. Ensure that the change you expected was propogated to the Project Monitoring Stack (e.g. Alertmanager is no longer deployed).

## Uninstalling Prometheus Federator

After deleting the Helm Charts, you may want to manually uninstall the CRDs from the cluster to clean them up:

```bash
## Helm Project Operator CRDs
kubectl delete crds projecthelmcharts.helm.cattle.io

## Helm Locker CRDs
kubectl delete crds helmreleases.helm.cattle.io

## Helm Controller CRDs
##
## IMPORTANT NOTE: Do NOT delete if you are running in a RKE2/K3s cluster since these CRDs are used to also manage internal k8s components
kubectl delete crds helmcharts.helm.cattle.io
kubectl delete crds helmchartconfigs.helm.cattle.io
```

> Note: Why aren't we packaging Helm Project Operator CRDs in a CRD chart? Since Helm Project Operator CRDs are shared across all Helm Project Operators (including this one), the ownership model of having a single CRD chart that manages installing, upgrading, and uninstalling Helm Project Operator CRDs isn't a good model for managing CRDs. Instead, it's left as an explicit action that the user should take in order to delete the Helm Project Operator CRDs from the cluster with caution that it could affect other deployments reliant on those CRDs.