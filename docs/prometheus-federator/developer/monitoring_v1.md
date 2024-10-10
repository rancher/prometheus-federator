### Disclaimer

**This is intended for internal developer use only and is not meant for broader consumption**.

**For user-facing docs, please consult the official Rancher docs.**

---

## [Legacy] What is Monitoring / Alerting V1?

Monitoring / Alerting V1 are the legacy solutions that Rancher offered as the default Monitoring / Alerting solution via the Cluster Manager UI up till Rancher 2.4.x.

In Rancher 2.5.0+, Cluster Monitoring / Alerting V1 has been deprecated in favor of a single Monitoring V2 solution, which also supports metrics-based Alerting.

In Rancher 2.6.5+, Project Monitoring / Alerting V1 has been deprecated in favor of a single Prometheus Federator solution, where each Project Monitoring Stack also supports Alerting.

This document will cover the major differences between Monitoring V1's offerings in comparision to Monitoring V2's offerings from a technical perspective. 

## High-Level Differences

### Source Code and Chart Type

While Monitoring V2 is located in the [`rancher/charts`](https://github.com/rancher/charts) repository solely as a single, self-contained chart, the old Monitoring V1 and Alerting V1 solution is split across [`rancher/system-charts`](https://github.com/rancher/system-charts) and a set of legacy controllers embedded into Rancher itself.

The fundamental difference between charts hosted in these repositories is that `rancher/system-charts` contains **Helm 2** charts, whereas `rancher/charts` contains **Helm 3 charts**. 

As a result, the charts that were contained in `rancher/system-charts` were packaged and deployed using Rancher's old Catalog V1 solution (which supported Helm 2 charts). This solution would embed a Helm 2 server ([Tiller](https://v2.helm.sh/docs/install/)) instance into Rancher itself and execute Helm client calls when necessary to install/upgrade/delete charts on behalf of the user (whenever an API call is made or some other situation necessitates it).

On the other hand, the charts that are contained in `rancher/charts` are packaged and deployed using Rancher's current Catalog V2 solution (also known as **Apps & Marketplace**, which only supports Helm 3 charts), which does not embed a Helm server of any kind. Instead, it simply creates Helm operation pods on behalf of the user, which contains a [`rancher/shell`](https://github.com/rancher/shell) container that simply runs the appropriate `helm` command after mirroring user permissions.

This is why Helm operation logs in Monitoring V1 are all directly found **in the main Rancher deployment's logs**, whereas Monitoring V2's Helm operation logs are tied to the Helm operation containers that get deployed.

### Enabling / Disabling Monitoring and Alerting

Monitoring V2 takes an imperative approach, where a user explicitly requests Rancher to perform **every** Helm operation, from `helm install` to `helm upgrade` to `helm delete`. You can also only deploy Monitoring+Alerting V2; there's **no independent Alerting solution offered in the V2 world**.

On the other hand, Monitoring / Alerting V1 follows a **declarative** approach; a user simply sets the spec fields for `enableClusterMonitoring` or `enableClusterAlerting` on the `cluster.management.cattle.io/v3` object (or sets `enableProjectMonitoring`  / `enableProjectAlerting` on the `project.management.cattle.io/v3` object for a given Rancher Project) and Rancher uses controllers to force the current state of the cluster's Monitoring deployments to match the desired spec.

> **Note**: This problem with this approach is that a failure state in executing a Helm operation would result in the Rancher logs **continously emitting failures** until a user fixes the issue that caused the problem.
>
> For example, if a user were to upgrade to a version of Kubernetes that Monitoring V2 is not supported in, they would get a single failure on trying to install Monitoring V2 that this version is not supported.
>
> Whereas if a user were to do the same in Monitoring V2 and set `enableClusterMonitoring: true`, this would result in constant failures until they went into the cluster and specifically set `enableClusterMonitoring` to false.
>
> In addition, this approach also fails whenever the underlying object containing the setting for whether Monitoring should be deployed or not is managed by another resource's configuration. 
>
> For example, on introducing the new provisioningv2 framework for Rancher to support RKE2 / k3s provisioning using a [CAPI](https://cluster-api.sigs.k8s.io), Rancher introduced a set of controllers that cause 4 different cluster objects to get created and synced in conjunction (a `v3` Cluster, a new `v1` Cluster, a [Fleet](https://github.com/rancher/fleet) cluster, and a [CAPI](https://cluster-api.sigs.k8s.io) cluster). Amongst these 4, either the `v1` Cluster would be the "parent" object (if the user created the cluster on the Dashboard UI) or the `v3` Cluster would be the "parent" object (if the user created the cluster on the Cluster Manager UI, or used legacy RKE1 provisioning).
>
> As a result, when the `v1` Cluster is the parent object, any change to the `v3` Cluster is no longer persisted, resulting in the **inability to enable Monitoring V1 in any cluster provisioned via the Rancher Dashboard UI**. On trying to still enable it, users would encounter flapping conditions where the `v3` object would be modified, try deploying the Helm 2 `rancher-monitoring` chart, and then immediately have that field disabled by Rancher controllers, resulting in Rancher immediately trying to uninstall the chart.

If any of the above values are set for any object, Monitoring V1 logic within Rancher will always deploy and keep installed Prometheus Operator in one chart release (using the **same underlying chart**; other components are just disabled).

Depending on the specific field enabled, Monitoring V1 logic within Rancher would then install either **only Prometheus+Grafana+Exporters** (if `enable<Project|Cluster>Monitoring` was set) or **only Alertmanager+Webhook-Receiver** (if `enable<Project|Cluster>Alerting`). If both are set, it would automatically install both components, but as separate chart releases.

Therefore, having both Monitoring V1 and Alerting V1 enabled results in a total of three charts releases:
- One for Prometheus Operator
- One for Prometheus+Grafana+Exporters
- One for Alertmanager+Webhook-Receiver

When only Alerting is enabled, unlike in Monitoring V2 which only supports **metric-based Alerting** (i.e. all alerts always come from Prometheus), Alerting V1 allows you to register non-metric-based alerts where Rancher directly pings Alertmanager. These alerts are triggered by the `*Watcher` controllers described [below](#rancher-monitoring-v1-crds-and-embedded-controllers).

### Underlying Components

From a chart design perspective, it's easier to compare Monitoring V1 against the upstream [`kube-prometheus-stack`](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack) chart, since they were developed in parallel around the same timeframe.

When both Monitoring V1 and Alerting V1 are enabled, Monitoring V1 deploys all the underlying components listed in [the Monitoring V2 docs around `kube-prometheus-stack`](monitoring_v2.md#what-does-it-deploy). 

The primary differences include that it:
- Has far fewer configuration options than upstream **(this is primarily why Monitoring V2 is based on an upstream chart instead of Monitoring V1)**
- Uses a completely different set of Rancher-original dashboards and alerts instead of upstream dashboards and alerts, which requires more maintainence from Rancher **(a secondary reason why Monitoring V2 is based on an upstream chart instead of Monitoring V1)**
- Supports deploying `wmi_exporter` (now called [`windows_exporter`](monitoring_v2.md#windowsexporterhttpsgithubcomprometheus-communitywindowsexporter-for-windows-support) in Monitoring V2, but it's the exact same codebase / solution) for RKE1 Windows Monitoring.
- Has default exporters for integration with Rancher's Logging V1 and Istio V1 System Charts
- Deploys [`rancher/webhook-receiver`](https://github.com/rancher/webhook-receiver), which is the Monitoring V1 equivalent of [Alerting Drivers](monitoring_v2.md#add-on-chart-alerting-drivers) with Rancher-maintained non-native notification providers for Alertmanager, as opposed to Alerting Drivers which entirely packages official upstream solutions.
- Uses [`rancher/prometheus-auth`](https://github.com/rancher/prometheus-auth) as a proxy in front of all Prometheus instances to ensure that users (or Grafana) are only able to see the values that they should have permissions to be able to see

> **Note**: How does Prometheus Auth work on a high-level?
>
> Since Prometheus Auth is deployed as a container in your Kubernetes cluster, the API requests contain [Bearer Authentication Tokens](https://swagger.io/docs/specification/authentication/bearer-authentication/) in their headers; this is the token used by the Kubernetes API to authenticate your request.
>
> By grabbing the token and using it to submit a [TokenReview](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-review-v1/), Prometheus Auth can get a struct representing the Kubernetes-authenticated user that is making the request (i.e. either a user or a ServiceAccount tied to Prometheus / Grafana). On receiving the user struct, Prometheus Auth detects whether that user struct matches its own user (which since the Prometheus and Grafana in Monitoring V1 uses the same ServiceAccount, identifies whether this request is coming from Prometheus or Grafana's backends). If this is the case, it allows the request to proceed without any hijacking; it's a direct proxy.
>
> However, if the user struct does not match the Prometheus / Grafana user itself, Prometheus Auth will check to see whether the token is tied to an existing Secret containing a ServiceAccountToken (i.e. a Project Prometheus); if it is, it will grab the namespace that Secret resides in and see whether the `project-monitoring` ServiceAccount in that namespace (which is the one tied to Grafana and Prometheus only for Project Monitoring stacks) allows the ability to `view` the resource `prometheus` in the `monitoring.cattle.io` API Group (note: this is not the normal `monitoring.coreos.com` API group; this is a dummy API group specifically to provide this permission that is added to all Project Member and Project Owners). This is performed via a [SubjectAccessReview](https://kubernetes.io/docs/reference/kubernetes-api/authorization-resources/subject-access-review-v1/) by Prometheus Auth.
>
> This will naturally be the case since, on deployment, Monitoring V1 ties the `project-monitoring-readonly` ProjectRoleTemplate to the `project-monitoring` ServiceAccount, which provides it that permission in all namespaces tied to the Project, so such a request will be authenticated. From here, the project ID label tied to the namespace that the secret was found in is used to index all namespaces whose metrics should be accessible from a given request.
>
> From here, now that the authentication / authorization pieces have been figured out, Prometheus Auth can simply receive the HTTP request, extract the PromQL query, modify the submitted query by adding `namespace~="<namespaces...">` in the query to filter by metrics only explicitly tied to Project namespaces, and then send over the "hijacked" query to Prometheus; it then forwards the response from Prometheus back to the Project Prometheus that made the request.

### Approach to Project Monitoring

In Monitoring V1, deploying Project Monitoring V1 happened in the same way that the overall cluster monitoring solution was deployed (via a field in the `Project` CR instead of the `Cluster` CR).

The only difference was that exporters would be disabled in favor of a single exporter that [federated](https://prometheus.io/docs/prometheus/latest/federation/) (i.e. treated another Prometheus as an Exporter itself) **all *queriable* metrics** from the Cluster Prometheus to the Project Prometheus (see [section below on Prometheus Auth and Project Monitoring](#multi-tenancy-with-project-monitoring-and-rancherprometheus-authhttpsgithubcomrancherprometheus-auth)).

> **Note**: Since Monitoring V1 leverages [`rancher/prometheus-auth`](https://github.com/rancher/prometheus-auth) as an authorization proxy for accessing the underlying Prometheus, "all *queriable* metrics" in Monitoring V1 is only the metrics that `rancher/prometheus-auth` would allow you to see after "hijacking" your request.

On the other hand, Prometheus Federator (Project Monitoring V2) has an entirely different way of being deployed than Monitoring V2 (leveraging [k3s-io/helm-controller](https://github.com/k3s-io/helm-controller) to deploy the Project Monitoring Stack, [rancher/helm-locker](https://github.com/rancher/helm-locker) to "lock" deployed stack and prevent any changes, and [rancher/helm-project-operator](https://github.com/rancher/helm-project-operator) as the overall orchestrator that handles watching `ProjectHelmChart` CRs and creating `HelmChart` and `HelmRelease` CRs on the resource's behalf). Project Monitoring V2 also handles federation directly; the Project Prometheus's federation configuration itself uses matchers to limit the metrics that can be collected from the Cluster Prometheus (which is directly accessed, as opposed to accesses via a [`rancher/prometheus-auth`](https://github.com/rancher/prometheus-auth) proxy)

More information on this architecture can be found in the [Project Monitoring docs](../design.md).

### Rancher Monitoring V1 CRDs and Embedded Controllers

In order to be fully functional, Monitoring V1 also required the use of controllers embedded in the Rancher UI.

While some of these controllers operate on native Kubernetes resources (such as the `ExporterEndpointController`), some controllers also sometimes directly operate / manage Prometheus Operator CRs

The issue with this appraoch is explained in more depth in a note in [the main Monitoring V2 docs around the CRD chart](monitoring_v2.md#crd-chart-with-install--uninstall-jobs), but essentially splitting ownership of Prometheus Operator CRs between Prometheus Operator itself (that is deployed by the Monitoring V1 chart) and Rancher creates a lot of edge conditions; this is why Monitoring V2 primarily moved towards a **decoupled** approach with respect to Rancher and Monitoring V2.

However, assuming that this doesn't cause an issue, here are the different controllers that get deployed for **Monitoring V1**:
- `ExporterEndpointController`: keeps track of all nodes in the cluster and manually maintains (injects into or removes endpoints from) the `Endpoints` objects tied to `etcd`, `kube-scheduler`, `kube-controller-manager`, and `node-windows`. These Endpoints objects were tied to headless Services deployed by the Monitoring V1 chart that the Monitoring V1 chart would already have `ServiceMonitors` pointing to, but unlike in Monitoring V2 where we have [PushProx](monitoring_v2.md#pushprox-exporters-for-most-kubernetes-internal-components) that handles discovering new targets itself via scheduling the clients onto those nodes and populating the `Endpoints` objects based on labelSelectors on workloads, this needed to be manually maintained by Rancher legacy controllers
- `ConfigRefreshHandler`: keeps track of which namespaces are in a current Project and triggers an update of a Project Monitoring Stack deployment if necessary.
- `MetricsServiceController`

In addition, Monitoring V1 creates two CRDs that are not managed by controllers: `ClusterMonitorGraphs`, which embed `MonitorMetrics`. Since Monitoring V1 did not really support user-added graphs onto Grafana like Monitoring V2, `ClusterMonitorGraphs` were a way for users to add additional graphs (in addition to the defaults) that would directly be interpreted by the Rancher UI and embedded onto the Cluster Manager UI itself; this is why on loading the Cluster Manager Cluster Detail page, you might see network calls be made for the Rancher UI to get these resources, followed by requests to Prometheus to collect metrics to populate the graphs that are created.

On the other hand, here are the different controllers that get deployed for **Alerting V1** and the CRDs that they manage:

- `ConfigSyncer`: converts **`<Cluster|Project>AlertRule` CRs** and **`<Cluster|Project>AlertGroup` CRs** into native Prometheus Operator **PrometheusRule CRs** when they target metric-based alerts and uses the information within then to configure the Monitoring V1's Alertmanager secret routing tree. Also converts **Notifier CRs** (the Monitoring V1 equivalent of `Receivers` in Monitoring V2, except they can only support exacty one notification configuration rather than one or more) into the Monitoring V1 Alertmanager secret receivers.
- `StateSyncer`: updates the status of **`<Cluster|Project>AlertRule` CRs** based on the current state of the corresponding alarm in the Alertmanager API; this allows Rancher to show an alarm as firing to a user without pinging the Alertmanager API every time a user refreshes a page
- `ClusterScanWatcher`, `EventWatcher`, `NodeWatcher`, `WorkloadWatcher`, `SysComponentWatcher`, `PodWatcher`: has Rancher watch a specified type of resource (`ClusterScans` from CIS v1, Kubernetes `Events`, nodes, workloads, Kubernetes system components, and `Pods`, respectively) and has Rancher trigger HTTP calls on behalf of the controllers to Alertmanager to trigger an alert / notification. The alerts themselves are configured by the configuration of **`<Cluster|Project>AlertRule` CRs** in the cluster that have been added by users

> **Note**: Conceptually, the `<Cluster|Project>AlertRule` is pretty similar to a `PrometheusRule` except it can only contain alerting rules (or the configuration necessary for non-metric based alerting configurations which would be picked up by the `*Watcher` controllers) and it additionally contains the route configurations that would apply if this alert were to be triggered.
>
> As a result, it's simpler to quickly create alerts that work since you don't need to separately configure the PrometheusRule with the definition of the alert and the Alertmanager secret with the information of what to do when you receive an alert (i.e. where to send it to); that's all encoded in one CR.

## Migrating From V1 To V2

Since the overall operating models of V1 and V2 from a user perspective are fundamentally different (especially around Alerting V1 -> V2, where users need to switch from working on a higher level with **`<Cluster|Project>AlertRule`, `<Cluster|Project>AlertGroup`, and `Notifier` CRs** to directly interfacing with the underlying `PrometheusRule` CRs and Alertmanager Configuration Secret), migrating from V1 to V2 is a **hard** migration. 

You effectively need to completely disable and remove your Monitoring V1 / Alerting V1 setups and then translate your `Notifier` CRs into Receivers in your new Alertmanager Secret for V2 and your `<Cluster|Project>AlertRule` / `<Cluster|Project>AlertGroup` CRs into both `PrometheusRules` and Routes in your new Alertmanager Secret for V2.

For the first task of completely disabling and removing your Monitoring V1 / Alerting V1 setups (including on a Project Level), a script in [`bashofmann/rancher-monitoring-v1-to-v1`](https://github.com/bashofmann/rancher-monitoring-v1-to-v2) exists called `check_monitoring_disabled.py` that can be provided a **Cluster Admin** Rancher API token and verify that a given Cluster identified by a given Cluster ID (i.e. `c-XXXXX`) has completely disabled Monitoring V1 across all Cluster and Project resources that are found in the Rancher management cluster.

For the second task, this repository also contains scripts like `migrate_dashboards.py` and `migrate_alerts.py` that can be used **before you remove Monitoring / Alerting V1 completely in the previous step** to automatically produce a Kubernetes manifest containing any user-added Grafana dashboards in Monitoring V1 and the underlying PrometheusRules that were created **only for metric-based alerts** for your Alerting V1 setup.

However, **these scripts are not generally recommended** since they will create resources that have script-generated names that are not as human-readable and can duplicate alerts and dashboards that are already being handled by the new Monitoring V2 default set of alerts and dashboards. It's generally a better idea to just identify what needs to be migrated and add them into a fresh Monitoring V2 setup the way that Monitoring V2 intends it, via the Rancher UI.
