# Getting Started

## Simple Installation

### In Rancher (via Apps & Marketplace)

1. Navigate to `Apps & Marketplace -> Repositories` in your target downstream cluster and create a Repository that points to a `Git repository containing Helm chart or cluster template definitions` where the `Git Repo URL` is `https://github.com/rancher/helm-project-operator` and the `Git Branch` is `main`
2. Navigate to `Apps & Marketplace -> Charts`; you should see a chart under the new Repository you created: `Helm Project Operator`. 
3. Install `Helm Project Operator`

### In a normal Kubernetes cluster (via running Helm 3 locally)

Install `helm-project-operator` onto your cluster via Helm to install the Helm Project Operator

```
helm install -n cattle-helm-system helm-project-operator charts/helm-project-operator
```

### Checking if ProjectHelmCharts work

1. Ensure that the logs of `helm-project-operator` in the `cattle-helm-system` namespace show that the controller was able to acquire a lock and has started in that namespace
2. Deploy a ProjectHelmChart into a Project Registration Namespace (see [design.md](design.md) for more information on how to identify this)
3. Check to see if a HelmChart CR was created on behalf of that ProjectHelmChart in the Operator / System (`cattle-helm-system`) namespace
4. Find the Job in the Operator / System (`cattle-helm-system`) namespace tied to the HelmChart object to view the Helm operation logs that were performed on behalf of the HelmChart resource created; these logs should show as successful.
5. Check to see if a HelmRelease CR was created on behalf of that ProjectHelmChart in the Operator / System (`cattle-helm-system`) namespace
6. Ensure that the status of the HelmRelease CR shows that it has successfully found the Helm release secret for the Helm chart deployed by the HelmChart CR.
7. Locate the Project Release Namespace (see [design.md](design.md) for more information on how to identify this) and ensure that the ConfigMaps contained within `charts/project-operator-example` were deployed onto the cluster.
8. Try to modify or delete the resources; you should see that they are instantly recreated or fixed back into place.
9. Try supplying overrides to the deployed Helm chart by modifying `spec.values` on the ProjectHelmChart resource the `data` value to any YAML you want; on supplying new YAML to the ProjectHelmChart, you should see the Helm Operator Job (deployed on behalf of the HelmChart resource) be modified and you should observe that the HelmRelease CR emits an event (observable by running `kubectl describe -n cattle-helm-system <helm-release>` on the HelmRelease object) that indicates that it is Transitioning and then Locked; the release number will also be updated.

## Uninstalling Helm Project Operator

After deleting the Helm Charts, you may want to manually uninstall the CRDs from the cluster to clean them up:

```bash
## Helm Project Operator CRDs
kubectl delete crds projecthelmcharts.helm.cattle.io

## Helm Locker CRDs
kubectl delete crds helmreleases.helm.cattle.io

## Helm Controller CRDs
##
## IMPORTANT NOTE: Do NOT delete if you are running in a k3s/RKE2 cluster since these CRDs are used to also manage internal k8s components
kubectl delete crds helmcharts.helm.cattle.io
kubectl delete crds helmchartconfigs.helm.cattle.io
```

> Note: Why aren't we packaging Helm Project Operator CRDs in a CRD chart? Since Helm Project Operator CRDs are shared across all Helm Project Operators (e.g. [rancher/prometheus-federator](https://github.com/rancher/prometheus-federator)), the ownership model of having a single CRD chart that manages installing, upgrading, and uninstalling Helm Project Operator CRDs isn't a good model for managing CRDs. Instead, it's left as an explicit action that the user should take in order to delete the Helm Project Operator CRDs from the cluster with caution that it could affect other deployments reliant on those CRDs.