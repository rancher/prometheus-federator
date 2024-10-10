### Disclaimer

**This is intended for internal developer use only and is not meant for broader consumption**.

**For user-facing docs, please consult the official Rancher docs.**

---

## What is Monitoring V2?

Monitoring V2 is Rancher's built-for-Rancher **single-cluster** Monitoring solution, packaged in a Helm chart named `rancher-monitoring`.

The chart itself is hosted in the latest `dev-v2.x` branch (or `release-v2.x`, for stable versions, which are also hosted on Rancher's Apps & Marketplace by default via `charts.rancher.io`) in [`rancher/charts`](https://github.com/rancher/charts) as a [`rancher/charts-build-scripts`](https://github.com/rancher/charts-build-scripts) Package, just like a **patched** copy of it is represented in this repository under [`packages/rancher-project-monitoring`](../../packages/rancher-project-monitoring/). 

The `rancher-monitoring` chart itself is patched from the upstream [`kube-prometheus-stack`](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack) chart; therefore, anything that applies to the `kube-prometheus-stack` chart at the version indicated by the `+up` annotation on `rancher-monitoring` versions (i.e. `102.0.0+up40.1.2` indicates the base of the chart is [`kube-prometheus-stack` at version `40.1.2`](https://github.com/prometheus-community/helm-charts/commit/89f19277f7afbbfb2d47b4f59dfecec2fd60f376)) usually applies to `rancher-monitoring` as well.

> **Note**: To figure out exactly what `rancher-monitoring` patches in the latest chart version, you can look at the contents of `packages/rancher-monitoring/*/generated-changes/<exclude|overlay|patch>`, which shows either files or .diffs of files that differentiate the chart from upstream; you will need to look through each `packages/rancher-monitoring/*/generated-changes/dependency` independently though.

## `kube-prometheus-stack`

[`kube-prometheus-stack`](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack) contains a collection of Kubernetes manifests, [Grafana](https://grafana.com/) dashboards, and [Prometheus](https://prometheus.io/docs/introduction/overview/) rules combined with documentation and scripts to provide easy to operate end-to-end Kubernetes cluster monitoring with [Prometheus](https://prometheus.io/docs/introduction/overview/) using the [Prometheus Operator](https://prometheus-operator.dev/).

### What does it deploy?

Mainly, it deploys:

- Prometheus Operator: the main manager of **anything related to Prometheus or Alertmanager**, but **not Grafana**
  - See `templates/prometheus-operator` in the chart
- A Prometheus CR, managed by Prometheus Operator: one-to-one with Prometheus workloads in your cluster
  - See `templates/prometheus` in the chart
- An Alertmanager CR, managed by Prometheus Operator: one-to-one with Alertmanager workloads in your cluster
    - See `templates/alertmanager` in the chart
- A set of default ServiceMonitor CRs, managed by Prometheus Operator: determines the Prometheus [scrape configuration](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#scrape_config)
    - See `templates/exporters` in the chart; also contained in other files usually titled `servicemonitor.yaml`
- A set of default PrometheusRules CRs reprenting [recording rules](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/) and [alerting rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/), managed by Prometheus Operator: determines the Prometheus rules file
    - See `templates/prometheus/rules-1.14` in the chart

To collect metrics by default, it deploys:
- An embedded [`node-exporter`](https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus-node-exporter) chart: collects metrics about the physical nodes themselves, deploys as a DaemonSet to all nodes
- An embedded [`kube-state-metrics`](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-state-metrics) chart: collects metrics from the Kubernetes API (only need 1 since there's only 1 API per cluster)

In addition, it deploys:

- An embedded [Grafana](https://github.com/grafana/helm-charts/tree/main/charts/grafana) chart that deploys Grafana alongside several sidecars (dashboards, datasources, etc.) that watch for ConfigMaps in the cluster
- A set of default Grafana Dashboard ConfigMaps, picked up by a Grafana sidecar: one-to-one with the Dashboards that show up on the UI for the Grafana instance that is picking up these dashboards
   - See `templates/grafana/dashboards-1.14` in the chart for the content and `templates/grafana/configmap-dashboards.yaml` for the ConfigMaps
- A default Grafana Datasource ConfigMap, picked up by the Grafana sidecar: sets up Grafana to query metrics from Prometheus to populate its visualizations
    - See `templates/grafana/configmaps-datasources.yaml` in the chart

And a host of other things you need around all of this (PersistentVolumeClaims, Ingresses, etc.). 

### Installing Monitoring

After installing `kube-prometheus-stack`, the following steps will happen in-parallel:
1. Node Exporter and Kube State Metrics are deployed and start running on the cluster. Metrics are being collected from the relevant targets (i.e. nodes, Kubernetes API) and are being exposed at `http://<component>.cattle-monitoring-system.svc:<port>/metrics` on each of their endpoints. **Metrics become ready to be scraped from each target whenever Prometheus is ready**.
2. Prometheus Operator and all its CRDs and custom resources (one `Prometheus`, one `Alertmanager`, many `ServiceMonitor`s, many `PrometheusRule`s, etc.) will be applied to the cluster. Prometheus Operator will spin up one Prometheus workload and one Alertmanager workload configured in accordance to the CRs. Alertmanager will initally come up with no alerts. Prometheus will initially come up with no targets, until Prometheus Operator modifies its scrape configuration to add all the targets from `ServiceMonitors` it has picked up and modifies its rules files to add all the rules from `PrometheusRules` it has picked up. At this point, all the targets will be listed in the Prometheus UI under `Targets` but show `Unknown` and Alertmanager will start to receive the `Watchdog` alert from Prometheus, which indicates that the alerting pipeline is up and functional. **Targets will show `Unknown` till Prometheus successfully scrapes each target, after which it will show `Up`, at which point Prometheus should start populating with data.**
3. Grafana will be deployed onto the cluster along with the Grafana Dashboard / Datasources ConfigMaps. Grafana will come up without any dashboards and datasources loaded. The Grafana sidecars will list all the ConfigMaps in the configured namespaces (or all namespaces) with the expected label and load them into Grafana. Grafana's UI will now have the Prometheus datasource and the dashboards loaded, but everything will show `No Data`. **Grafana dashboards will show data whenever Prometheus's API is up and enough metrics have been collected by Prometheus for all the PromQL queries in the dashboards to work**.

### The Lifecycle of a Metric

Once enough time has passed, we should then see the following process continually occur:
- Exporters talk to systems (hardware & OS metrics for `node-exporter`, Kubernetes API for `kube-state-metrics`) and "export" their metrics by serving metrics in a valid [exposition format](https://prometheus.io/docs/instrumenting/exposition_formats/) at an endpoint. For example, metrics may be served at `http://localhost:9796/metrics`.
- Prometheus performs scrapes based on targets in its scrape configuration on a routine interval. On a successful scrape, it records each of the series listed at the endpoint from that scrape in memory and stores each series somewhere in a [128MB write-ahead log (WAL) segment](https://prometheus.io/docs/prometheus/latest/storage/), a file on their filesystem that stores the metrics from the last 2 hours (to prevent loss of data on a program crash). The data from the WAL is eventually compacted into blocks every 2 hours, which is then persisted into the `chunks` directory. The data is now queriable on the Prometheus Graph UI.
- In parallel:
  - Prometheus executes recording rules based on rules in its rules files on a routine interval. Either more synthetic series are created (via recording rules that execute PromQL expressions and create series based on those expressions) or Prometheus fires an alert to Alertmanager. On receiving an alert, Alertmanager consults its in-memory state whether a notification needs to be sent, passes the received alert through the routing tree specified in its configuration, and fires an alert to all matched notification configurations.
  - On a user first accessing the Grafana UI or every time they refresh the page, Grafana will make one or more requests to Prometheus based on each dashboard's underlying [PromQL query](https://prometheus.io/docs/prometheus/latest/querying/basics/), each of which Prometheus will compute and return back with whatever data it has collected so far. Grafana will then render the visualizations based on the styles provided in the Grafana Dashboard JSON.

> **Note**: Prometheus cannot serve as a standalone alerting system because its alerting is **stateless**, as opposed to Alertmanager which is **stateful**. 
> 
> As a result, Alertmanager can intelligently send alerts on user-defined periodic intervals, not on a specific rule evaluation interval, and intelligently notify users that alerts have been resolved only if the last conveyed message to the user was that the alert was triggering.
>
> Prometheus, on the other hand, can only continously send alerts, since it doesn't store past context.
>
> In addition, Alertmanager supports several native [receivers](https://prometheus.io/docs/alerting/latest/configuration/#receiver) that it can send alerts to based on a [route tree](https://prometheus.io/docs/alerting/latest/configuration/#route) which determines where alerts go based on matching labels and annotations on alerts.

> **Note**: Prometheus cannot serve as a standalone visualization system because it does not persist dashboards; the primary purpose of Grafana in this stack is to allow users to persist dashboards, which is easier than copying / pasting those dashboards into the Prometheus UI on first loading the page.
>
> In addition, Grafana offers far more customization its visualizations.

> **Note**: Since Grafana makes all these queries to the underlying Prometheus every time a user refreshes their page, it's usually a good idea to make any overly complex Grafana expressions into recording rules. That way, you put less strain on Prometheus on users refreshing their pages, since the requests don't require additional computation.

### Debugging Monitoring

An effective way to debug a bug in Monitoring is to reproduce the setup and then backpedal through [the installation process](#installing-monitoring) or [the lifecycle of a metric](#the-lifecycle-of-a-metric) from the component that is exhibiting the failure through all dependencies that affect that behavior; that way, you can identify if the reason why you are seeing an issue is because of the component that is exhibiting the symptoms (i.e. Grafana, Alertmanager) or something it depends on is not healthy (i.e. `node-exporter`, `kube-state-metrics`).

On a high level, here is the list of every component and its dependencies:
- **Exporters (Node Exporter, Kube State Metrics)**: No dependencies. Should be working / testable standalone.
- **Prometheus Operator and CRs**: No dependencies. Should be working / testsable standalone.
- **Prometheus**: For the Prometheus configuration (scrape configuration, rules files), it depends on **Prometheus Operator (via ServiceMonitors, PodMonitors, and Probes)**; therefore, if Targets or Alerts or Rules are missing on the Prometheus UI or the configuration looks wrong on the UI, you should look into Prometheus Operator. For metrics, it depends on all configured **Exporters**; therefore, if Targets are down or if data is missing when running a PromQL query on the Prometheus Graph UI, you should look into whether Exporters are providing metrics at their designated endpoints and whether Prometheus has the ability to reach the exporters based on how your cluster's networking is configured. For the workload configuration (i.e. `StatefulSet` or `Deployment` spec), it depends on the **Prometheus CR**; therefore, if it's missing something like a `PodSecurityPolicy` that grants it permission to be deployed or there's something wrong with a `PersistentVolumeClaim`, you should investigate the [`PrometheusSpec` fields](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/api.md#prometheusspec).
- **Alertmanager**: For Alertmanager configuration (routes and receivers), it depends on an **global Alertmanager secret `cattle-monitoring-system/alertmanager-rancher-monitoring-alertmanager`** and **Prometheus Operator (via AlertmanagerConfigs, which can provide *namespaced* alerting configuration that is added onto the global configuration)**; if Alerts that are showing up on the Alertmanager UI are not being sent to your configured notifiers or being sent to the wrong notifiers, you should look into how to Alertmanager configuration is generated from these sources. For actually receiving alerts, Alertmanager depends on **Prometheus**; if Prometheus is not firing an alert or if Prometheus cannot reach Alertmanager (due to networking issues), Alertmanager will not show an alert on the UI. For the workload configuration (i.e. `StatefulSet` or `Deployment` spec), it depends on the **Prometheus CR**; therefore, if it's missing something like a `PodSecurityPolicy` that grants it permission to be deployed or there's something wrong with a `PersistentVolumeClaim`, you should investigate the [`AlertmanagerSpec` fields](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/api.md#monitoring.coreos.com/v1.AlertmanagerSpec).
- **Grafana**: For its configuration (dashboards, datasource), it depends on the **Grafana sidecar containers** and **Grafana ConfigMaps**. If you see dashboards don't even populate on Grafana (i.e. you see literally nothing, not `No Data`), check the **Grafana dashboard sidecar** (specifically its logs and what namespaces it's configured to watch based on environment variables) and **persisted Grafana ConfigMap dashboards** (to ensure they got deployed onto the cluster in the first place). If you see all the dashboards but you see `No Data`, first check the Grafana UI to see if Prometheus is configured as a datasource (which can be checked in the settings after logging in to Grafana); if you don't see the datasource, check the **Grafana datasources sidecar** and the **persisted Grafana datasource dashboard**. For actually populating dashboards, Grafana depends on **Prometheus**; if Prometheus is not collecting the underlying metric for some reason or Prometheus is not accessible by Grafana due to how your cluster's networking is configured, it's possible that Grafana won't be able to make the calls to populate its visualizations.

Here are some general tips that apply to all Monitoring workloads, independent of what role they take in the stack:

1. If the chart is experiencing any issues related to Helm operations:
  - If the issue is with a `helm install` or `helm upgrade` before Helm even tries to apply resources onto the cluster, ensure that a basic `helm template` and `helm install --dry-run` works with the `values.yaml` overrides you've supplied to the command
  - If the Helm operation (`helm install|uninstall|upgrade`) is hanging, check to see if any [Helm hooks](https://helm.sh/docs/topics/charts_hooks/) that are `Jobs` that are still running on the cluster and see what the issue might be (see `When workloads aren't working as expected` below for more details on what to debug for those `Jobs`).

> **Note**: When Helm runs a Chart Hook for the Monitoring chart when installing / upgrading / uninstalling the Monitoring chart, Helm will only check to see whether the `Job` was successfully deleted to progress with the Helm operation. Therefore, if you manually delete the Job that the Helm operation is hanging on, Helm will proceed with the installation and report a successful installation, even though the Job didn't actually successfully finish running as it was prematurely deleted.
>
> This is not recommended in production setups, but can be a useful "hack" to debug such issues to see what else might be broken after a faulty Job `hook` has been deleted.

2. When workloads aren't working as expected:
  - **If the workload doesn't even exist (except Prometheus and Alertmanager)**: run a `helm template` locally for the rancher-monitoring chart with the given `values.yaml` and ensure that the resource exists in the manifest
  - **If there are missing or no Pods in the workload**: get the workload's YAML and check the `nodeSelectors` and `tolerations` to ensure they are as expected. If it seems like the Pod should be getting deployed onto this node, it's probably an admissions issue; you should look to see if your workload needs something like a `PodSecurityPolicy` or your release namespace needs a `Pod Security Admission` label to allow it to deploy. It's also possible that you are using something like `PersistentVolumes`, in which case a claim may not be being provided to you due to some issue with the underlying PV provider.
  - **If there are unready or unhealthy `initContainers` or `containers` in the workload**: Look at every log in every container of the workload. Note any anomalous logs down along with the component it was found from. Ensure that your workloads have the RBAC necessary to allow those containers to finish setup (i.e. Grafana sidecars have permissions to get and list `ConfigMaps` in the required namespaces), if they rely on the Kubernetes API. If you are testing a change to the configuration, check whether additional or different environment variable values need to be provided to the process to allow it to deploy as expected. In rare cases, it's also possible this could be a networking issue; if your container needs to do something like reach out to an external service, ensure that external service is up and use a command like `ping` or `wget` to ensure that your Pod can talk to that other Pod or service. If your container needs to reach out to an internal service in your cluster (i.e. Prometheus needs to reach out to Exporters), ensure that internal service is up and use a command like `ping` or `wget` to ensure that your Pod can talk to that other Pod or service (if not, it may be that you don't have the `NetworkPolicies` available to allow that communication to happen or, if it's a `hostNetwork` Pod like `rancher-pushprox` has for clients, it may be a node-level firewall issue that requires modifying the cloud-provider-level Network Security Groups attached to your nodes). In rarer cases, it's possible this could be a memory or CPU issue; check to see if the container was killed due to OOMKilled (out of memory killed) by running a `kubectl describe` on the workload and investigate whether the startup process may be resulting in large amounts of memory being generated (i.e. Prometheus trying to load up way too many WAL segments on initialization).

Here are some component-specific tips for debugging Monitoring:
1. Once Monitoring has been successfully installed, on the Kubernetes API or the Rancher UI, identify if every workload associated with Monitoring has been deployed and is healthy (from the perspective of Kubernetes). Also identify that there is actually a Pod tied to each of those workloads and the expected nodes have a Pod deployed (i.e. all nodes should have `node-exporter`, only one node should have `kube-state-metrics`, etc.)
2. For **each component** that is not healthy and the initial component that is exhibiting the issue, run through the checklist in `When workloads aren't working as expected`. Then:
  - **If your component is an Exporter (Node Exporter, Kube State Metrics)**: investigate whether you can execute into the Prometheus Pod (or any Pod if the Prometheus Pod is unavailable) and run a `wget` command to the configured endpoint via Kubernetes DNS (i.e. `https://<component>.cattle-monitoring-system.svc:<port>/metrics`). See if a valid set of metrics is emitted or whether you get a bad reponse. If that works and there's some other issue, check to see if there are any arguments or environment variables that need to be passed in. If you get a bad response or can't access the endpoint, it's likely a networking issue associated with either node-level firewall rules or Kubernetes-level `NetworkPolicies`
  - **If your component is Prometheus**: Go to the Targets page; if any Targets are down, go to the section above `If your component is an Exporter` for each target down. Go to your Alerts; ensure that only the `Watchdog` alert is firing. If any other alerts are firing, go to the Graph UI and run the PromQL expression associated with the alert; from there, you can modify the PromQL expression to glean more and more context until you can find the component that is causing the issue. If you find that metrics that you expect are missing, identify what Exporter is giving you that expected metric and go to the section on `If your component is an Exporter` for that Exporter. If the issue is a large amount of CPU / Memory consumption causing Prometheus to periodically restart, please see the section below on [Debugging Resource Spikes](#debugging-resource-spikes).
  - **If your component is Alertmanager**: If no alerts are firing, check to see if Prometheus is up; if it is not, go to the section above `If your component is Prometheus`. If the WatchDog alert is firing but some other alert is also firing that is unexpected, check to see what it might be indicating; it's possible that it's a valid error that's surfacing something wrong with another component. If an alert is firing that is expected, but no notification is being sent, first identify what labels and annotations it has. Then start at the top of the routing tree defined in the Alertmanager configuration (accessible via the Alertmanager UI) and see whether the configuration leads your alert through the expected path. If it does not, there might be missing labels or annotations that would indicate something that needs to be fixed on a Prometheus level. It may also be a Prometheus Operator issue if the configuration on the Alertmanager UI page was not what you expected; go to the section `If your component is Prometheus Operator`  if that's the case. If your configuration and alert both look good but the alert still does not seem to be sending, mimic similar alerts by directly making an HTTP request to the Alertmanager API with a simulated alert, following the format Alertmanager expects [in its client documentation](https://prometheus.io/docs/alerting/latest/clients/) (an [example](#example-send-alerts-to-alertmanager) is provided below). Finally, if all else fails check to see if the Alertmanager workload itself is improperly configured; to do this, check the Alertmanager CR to see if the `helm template` used for this release produced the right arguments on the CR. If the Alertmanager CR is properly configured, go to the section `If your component is Prometheus Operator`.
  - **If your component is Grafana**: If dashboards are missing, check if the Grafana dashboard sidecar has any error logs or issues or if it has incorrect or missing arguments / environment variable values. If the dashboards show `No Data`, login to Grafana (`admin`: `prom-operator` by default) and check to see if the datasource for Prometheus has been loaded and points to the right endpoint in the Grafana admin settings. If they still show `No Data`, check to see if you can exec into your Grafana pod and run a `wget` command on Prometheus; if you can't, there's probably an issue associated with networking (i.e. node firewalls are up, `NetworkPolicies` are used, etc.). If you can access Prometheus, check if it is healthy; if not, go to the section on `If your component is Prometheus`. If you find that some dashboards have data and some don't, look at the dashboards without data and inspect the JSON that is used to define its panel (you can access this on a dashboard if you are logged in on Grafana). Find all the PromQL expressions on that dashboard and run them on Prometheus's Graph UI; while investigating, go to the section on `If your component is Prometheus`.
  - **If your component is Prometheus Operator**: Most errors that have been identified as Prometheus Operator errors are likely exhibiting symptoms at a Prometheus or Alertmanager level; for issues that come from either of those levels, you should get all the Prometheus Operator CRs associated with the bad configuration and 1) ensure that the contents of the CR match what you expect to be deployed onto the cluster (i.e. a `helm template` rendering produced the expected result) and 2) check that the "translation" done by the Operator between all of the relevant CRs to configuration file used by the component (i.e. Prometheus scrape configuration, Prometheus rules files, Alertmanager configuration, etc.) was done correctly. If there's some other issue, see if there are any arguments or environment variables you need to pass in.
  - **If the error is observed on attempting to apply a Prometheus Operator CR (i.e. a `ServiceMonitor`) or executing a `helm upgrade`**: this may be due to the fact that Prometheus Operator serves as an admission webhook for its custom resources, which means that the Kubernetes API will reach out the Prometheus Operator after authorizing a request before accepting and fulfilling the request. This type of issue is common in upgrades since the Kubernetes API cannot execute apply operations on Prometheus Operator CRs if Prometheus Operator is down, and it may happen to be down during an upgrade.

### Debugging Resource Spikes

> **Note**: There's a fantastic [series of blog posts](https://ganeshvernekar.com/blog/prometheus-tsdb-the-head-block/) by an engineer in Grafana Labs that explains the internal workings of the Prometheus TSDB (time series database); it's worth reading this series to get an in-depth understanding of exactly how metrics are consumed by Prometheus from a TSDB perspective, since this can allow you to understand how to use something like `promtool` or how to design a custom script to investigate a Prometheus TSDB issue when you don't have access to the Prometheus UI. It can also help you understand what are good defaults to set for Prometheus command line arguments for a specific user's use case; for example, you may want smaller block sizes.

Since the core of the Monitoring stack is Prometheus, understanding the way that Prometheus stores metrics as a time series database is essential to understanding the necessary resource requirements to run it.

While it's possible that a flood of query requests (such as many users opening Grafana and constantly refreshing the page with complex queries that require Prometheus to touch a lot of blocks in the TSDB, such as metrics over long periods of time) can result in spikes to CPU usage and while growing memory usage is natural without a user using a solution like [Thanos](https://thanos.io/) or [Cortex](https://cortexmetrics.io/) for long-term storage of metrics outside of Prometheus's memory or the filesystem Prometheus is running on, generally what results in Prometheus being out of memory is ingesting a lot of **high cardinality metrics**.

To explain what that means, let's look at the general way that Prometheus ingests **Samples** to populate different **Series** from the response of a single Exporter to a single scrape request

Let's say our exporter reports the following metric: `pod_info`, which has labels for the Pod's name (`pod`) and the node it is on (`node`) and whose value is `1` if the pod exists. 

In Prometheus terms, our metric name is `pod_info` and `["pod", "node"]` are a set of labeled dimensions associated with this metric; together with a **defined** value for both labels, something like `pod_info{pod="a",node="1"} (1 timestamp) (2 timestamp)` forms what Prometheus calls a single Series: a set of Samples associated with a specific metric name across a set of labeled dimenensions.

An Exporter's response to scrape requests will be a set of multiple of these Series containing exactly one Sample (note: the timestamp is inferred from the scrape time). For example, you might see something like this at an Exporter's endpoint:

```promql
pod_info{pod="a",node="1"} 1
pod_info{pod="b",node="2"} 1
pod_info{pod="c",node="3"} 1
pod_info{pod="d",node="4"} 0
```

On receiving this scrape response, Prometheus will persist **4 different Series** to its WAL (and even internally in chunks, once they have been compacted); although they track the same **metric**, they are considered different Series since they have different values for their labeled dimensions. This is an issue because **Prometheus's TSDB is optimized to store a small set of Series with a growing / large set of samples; it is highly ineffecient with dealing with a large number of Series**.

The reason why storing multipe Series is problematic is that each Series takes up its own row in the TSDB, whereas each Sample only adds on its value and timestamp to a pre-existing row tied to a known Series, unless the Series has never beem seen before.

As a result, when Prometheus setups experience an explosion in the number of Series that need to be collected instead of the number of Samples due to some condition observed in a cluster that causes an Exporter to rapidly produce new values for labels, the resource requirements can heavily spike upwards.

As to why a cluster might be experiencing this, it's hard to say. Since a default Monitoring stack uses `kube-state-metrics` and `kubelet` metrics, a common cause for this type of explosion is for a cluster to experience "churn"; that is, the rapid creation and deletion of objects from the Kubernetes API, especially workloads, due to an issue with something like a controller that is in a failure loop, since this would result in new label values for fields like `pod` or `container` to be rapidly generated.

The only fix that has been identified to this type of issue is the increase resource memory requirements till Prometheus stabilizes and to prevent what caused the churn in the first place from happening.

## Rancher Monitoring

Now that we've discussed `kube-prometheus-stack`, which is the core / foundation of Monitoring V2, here are the **major** design differences where Monitoring V2 does something differently than upstream.

> **Note**: Since there are too many changes to describe manually, this guide will omit the minor changes, such as ensuring every workload has Linux / Windows `nodeSelectors` and `tolerations` and automatically prepending the value of `.Values.global.cattle.systemDefaultRegistry` to all images deployed by the chart for private registry support.

### PushProx Exporters for (Most) Kubernetes Internal Components

While Rancher previously had [Port Requirements](https://ranchermanager.docs.rancher.com/getting-started/installation-and-upgrade/installation-requirements/port-requirements) that would require all nodes of a Kubernetes cluster to expose the metrics ports in their firewalls (i.e. Network Security Groups or equivalent in a cloud environment) for Kubernetes internal components to all other nodes in the cluster, in Monitoring V2 we leverage a chart that deploys a forked version of [`prometheus-community/PushProx`](https://github.com/prometheus-community/PushProx) called [`rancher/PushProx`](https://github.com/rancher/PushProx). This is packaged into a Helm chart that lives in the latest branch of [`rancher/charts`](https://github.com/rancher/charts).

As described in the [Rancher docs](https://ranchermanager.docs.rancher.com/v2.6/integrations-in-rancher/monitoring-and-alerting/how-monitoring-works#how-pushprox-works) (with diagrams), PushProx is an **HTTP (Layer 7)** Proxy that allows Prometheus to send scrape requests that will be executed by PushProx Clients that make connections to the Proxy and relay Prometheus scrape requests to the desired target.

Rancher Monitoring heavily leverages PushProx to monitor Kubernetes internal components by placing clients in `hostNetwork` containers on each applicable node per component that will connect with the proxy and execute the scrape requests desired by Prometheus within the node's host network.

Rancher Monitoring contains **individual sets of PushProx proxy/clients for each Kubernetes distribution it supports**, so you will find that the chart has `<rke|rke2|kubeAdm><Etcd|Scheduler|ControllerManager|Proxy>` and `k3s<Server|Agent>` (where `k3s` packages all the components except `kubelet` in the single `k3sServer` binary, which emits metrics for all the internal components at one port). There are also some more such as `rke2IngressNginx`, `hardenedNodeExporter`, etc. that are available in the chart.

Since the components are placed on nodes via `nodeSelectors` on a PushProx client `DaemonSet`, the discovery of nodes that need to be monitored is handled by the Kubernetes scheduler itself. Advanced options can be provided to better select targets; see `rke2IngressNginx` in the `values.yaml` of the `rancher-monitoring` chart to see complex example where even a `Deployment` can be targeted and `affinity` is used instead.

Additional certificates can also be added on scrape using hostPath credentials by providing it to the clients; this is done in `rkeEtcd` in the `values.yaml` of the `rancher-monitoring` chart.

This chart also features the usage of a special `kubeVersionOverrides` Helm helper function that allows `values.yaml` overrides to be provided whenever the Kubernetes version of the current cluster matches the semver range provided. This allows the default `values.yaml` to encode overrides for seamless upgrades across Kubernetes versions. For an example, see `rke2IngressNginx` in the `values.yaml` of the `rancher-monitoring` chart.

> **Note**: The only Kubernetes component that we do not deploy a PushProx by default to is `kubelet`, although users can enable Monitoring `kubelet` via PushProx too by using the built-in `hardenedKubelet` chart.

> **Note**: `hardenedNodeExporter` also exists as a PushProx Exporter for `node-exporter` in the Monitoring chart. The only reason why a user should use this is if they aren't following the [Port Requirements](https://ranchermanager.docs.rancher.com/getting-started/installation-and-upgrade/installation-requirements/port-requirements#commonly-used-ports) listed in the Rancher docs, which expect that port `9796` is accessible across nodes in a Kubernetes cluster. Also, even if this exporter is enabled, Node Exporter still needs to be deployed to generate the underlying metrics that will be made accessible by PushProx.

> **Note**: Instead of using a **Layer 7 Proxy** like PushProx, it would be more ideal if the Monitoring chart were to use a **Layer 4 (TCP) Proxy** that would just allow you to forward the packets to Prometheus rather than reconstructing the HTTP request. This can be done by using [`rancher/remotedialer`](https://github.com/rancher/remotedialer), the same underlying technology that allows Rancher to communicate to airgapped clusters by having airgapped cluster form a reverse tunnel with the Rancher instance to allow packets to be forwarded when required, and an initial attempt of this has been made at [`aiyengar2/remoteproxy`](https://github.com/aiyengar2/remoteproxy).
>
> This would, however, remove the ability for PushProx to allow mounting `hostPath` certificates onto requests (which is required for monitoring RKE etcd nodes), since it would not be reconstructing requests that are sent over the wire.

### CRD Chart With Install / Uninstall Jobs

Generally, as listed in the [Helm docs](https://helm.sh/docs/chart_best_practices/custom_resource_definitions), there are two approaches that Helm charts can take with respect to deploying Custom Resource Definitions:
1. Expect a user to install the CRDs before installing the main chart **(this is the expectation of `kube-prometheus-stack`)**
2. Package a separate Helm chart that packages your CRDs **(this is the expectation of `rancher-monitoring`)**

Rancher Monitoring packages a separate CRD chart is because we expect users to primarily install Rancher Monitoring via the Apps & Marketplace Rancher UI, which uses the `catalog.cattle.io/auto-install` annotation on the `Chart.yaml` of `rancher-monitoring` to handle seamlessly installing the CRD chart before the main chart when a user requests an install or upgrade. 

> **Note**: Users will have to manually uninstall the CRD chart to remove it, however, although this is not recommended since it can break other applications using those CRDs such as Prometheus Federator.

This is currently set to `rancher-monitoring-crd=match`, which instructs Apps & Marketplace to find and install a chart named `rancher-monitoring-crd` whose version is identical to that of the current chart and install it onto the cluster before installing `rancher-monitoring`.

In addition, the main chart `rancher-monitoring` has `catalog.cattle.io/provides-gvr: monitoring.coreos.com.prometheus/v1`, which also affects the matching logic that Apps & Marketplace uses to find and install a chart that satisfies its requirements. This was originally used for Istio to match against ensuring Monitoring was installed first; however, it's not clear if there's any chart that utilizes this annotation's value today.

However, **unlike normal CRD charts in Rancher (which just directly package CRDs in the `templates/` directory**, Monitoring also utilizes a pre-install / pre-upgrade Job and post-delete Job to install and remove CRDs from the cluster.

This is necessary since Rancher has legacy controllers for the old [Monitoring V1](monitoring_v1.md#legacy-what-is-monitoring--alerting-v1) solution embedded within it, so on initializing the management agent controllers when Monitoring V1's feature flag is enabled **(even if Monitoring V1 is not currently installed in the cluster)**, Rancher creates the Prometheus Operator CRDs within the cluster before starting the controllers.

This poses a problem since Helm **cannot naturally assume ownership of non-Helm resources** (which gives a familiar error that says that it cannot import an object into a release due to missing labels), so instead Helm simply deploys a Job that runs `kubectl apply` and `kubectl delete` to manage the CRDs on Helm's behalf.

> **Note**: Users are recommended to shut off the feature flag for Monitoring V1 and restart Rancher controllers on upgrading from V1 to V2 of Monitoring since Rancher managing CRDs and potentially enabling Monitoring V1 controllers can impact Monitoring V2.
>
> Specifically, if there is drift between the versions of Prometheus Operator imported by Rancher and packaged into Monitoring V2, there's a possibility that a user can encounter extremely tricky edge cases around CRD fields. 
> 
> For example, in one issue encountered for the Monitoring V2 chart, a new version of Prometheus Operator was imported into Monitoring V2 but not into Rancher. So when the new version of Monitoring V2 was deployed onto a Rancher cluster, the user was able to see that the Prometheus CRD contained the field that they desired. But on running a `kubectl apply` to modify a Prometheus CR to have a non-default value for that field, they would see that while the `kubectl apply` was successful, the field would disappear on a `kubectl get`.
>
> This was because running the `kubectl apply` caused Rancher controllers to run their `OnChange` operation on the Prometheus CR, which requires marshalling the cluster's Prometheus CR into the Prometheus struct imported into the Rancher code and then unmarshalling the modified Prometheus struct into the cluster; however, since the Prometheus struct in Rancher did not contain that field (as it was based on an older version of Prometheus Operator), the field's non-default value would immediately be lost on completing the `OnChange` operation. 
>
> Bumping the Prometheus Operator version in Rancher solved this particular problem.

### Integration with Apps & Marketplace via Chart Annotations

Every Rancher chart has the following annotation:
- `catalog.cattle.io/certified: rancher`: whether a chart is certified by Rancher.

Most Rancher charts will also have:
- `catalog.cattle.io/release-name: rancher-monitoring`: the release name to use when installing this chart. Omitted when a chart can be installed onto multiple namespaces (i.e. [Alerting Drivers](#add-on-chart-alerting-drivers)), which will force a user to select a release name on install.
- `catalog.cattle.io/namespace: cattle-monitoring-system`: the namespace to install this chart. Omitted when a chart can be installed onto multiple namespaces (i.e. [Alerting Drivers](#add-on-chart-alerting-drivers)), which will force a user to select a namespace on install.

Whenever we have a chart that needs to exist in the Chart Repository since it needs to be installable but isn't shown to users (i.e. CRD charts), we normally add the following annotation:
- `catalog.cattle.io/hidden: "true"`: indicates this chart shouldn't show up in the Apps & Marketplace UI as a standalone chart for install. You can still find this chart's installation page, however, by adding a URL parameter to Rancher to show the hidden chart or directly navigating to the install page of the chart (i.e. start in Rancher Monitoring, replace the chart name in the URL with `rancher-monitoring-crd`, and you will get to the standalone page of the hidden `rancher-monitoring-crd` chart).

Whenever we have a chart that we built [a custom UI component within `rancher/dashboard`](https://github.com/rancher/dashboard/tree/master/shell/chart/monitoring) for, we usually add the following additional labels:
- `catalog.cattle.io/type: cluster-tool`: an annotation that indicates that this chart is supposed to be treated as a Cluster Tool by the UI.
- `catalog.cattle.io/display-name: Monitoring`: the display name to use for this chart on the Apps & Marketplace UI.
- `catalog.cattle.io/ui-component: monitoring`: the name of the component in the UI that renders the installation UI for this chart. The UI team will usually give this to you.

To control which clusters with Apps & Marketplace should see certain charts based on certain qualities of the cluster (such as the Rancher version managing it, the Kubernetes version it is running, whether it is a Windows cluster, etc.), you use the following annotations:
- `catalog.cattle.io/kube-version: '>= 1.16.0-0 < 1.26.0-0'`: A semver contraint string representing acceptable versions of the cluster's Kubernetes version to perform a **fresh install** of this chart onto. This field is not consulted on upgrades, unlike the default `Chart.yaml` field `kubeVersion`, which would fail to even render on a cluster with the wrong version. This is important since using the Rancher `kube-version` annotation would allow a user to upgrades to an unacceptable version of Kubernetes to still continue to run their existing Monitoring chart temporarily and perform in-place upgrades if necessary, till they are able to find a compatible version of the Monitoring chart released by Rancher, v.s. adding this constraint to the Helm `kubeVersion` annotation would break all in-place upgrades on a Kubernetes version upgrade of the underlying cluster.
- `catalog.cattle.io/rancher-version: '>= 2.7.0-0 < 2.8.0-0'`: A semver constraint string representing acceptable versions of the Rancher managing this cluster to perform a **fresh install** of this chart onto. Same logic as `catalog.cattle.io/kube-version` with respect to in-place upgrades.
- `catalog.cattle.io/permits-os: linux,windows`: A comma-delimited list of OSs (currently only `linux` and `windows` are accepted) that this chart is **allowed** to be deployed on without breaking. For example, if your chart is missing nodeSelectors or tolerations for Windows nodes since it expects itself to be deployed in a Linux-only cluster, the value of this annotation should be `linux` only. The Monitoring chart supports being deployed into Windows clusters, so it has both.

To emit certain warnings based on the qualities of the cluster that Monitoring is being installed onto, you use the following annotations:
- `catalog.cattle.io/requests-cpu: 4500m`: the minimum CPU that needs to be available in the cluster to install this chart. You can proceed with an install without this, but Apps & Marketplace will warn you before proceeding via a banner.
- `catalog.cattle.io/requests-memory: 4000Mi`: the minimum memory that needs to be available in the cluster to install this chart. You can proceed with an install without this, but Apps & Marketplace will warn you before proceeding via a banner.
- `catalog.cattle.io/deploys-on-os: windows`: A comma-delimited list of OSs (currently only `linux` and `windows` are accepted) that this chart **specifically has fields to deploy OS-specific components for (i.e. `global.cattle.windows.enabled`)** that may require the chart to be manually redeployed to attain full functionality if nodes of that type are added to the cluster. Primarily, this annotation results in the Apps & Marketplace UI showing the user a warning on detecting added nodes of that type (i.e. Windows servers) to ask a user to trigger an upgrade if they haven't done so yet.

Miscellaneous annotations:
- `catalog.cattle.io/upstream-version: 19.0.3`: indicates the verison of the upstream chart that is tied to this release of the chart in the `rancher/charts` Chart Repository.

### Nginx Proxy For Prometheus, Grafana, and Alertmanager

Since Rancher serves Prometheus and Grafana from a subpath of the Rancher URL, the Prometheus and Grafana UI can encounter issues where the Javascript loading from the Rancher URL fails since it cannot identify the right origin to use including the subpath.

To solve this problem, each component has a [NGINX](https://www.nginx.com/resources/glossary/nginx/) server container that serves as the main target for the component and proxies the requests received to the Prometheus / Grafana container running at a different port within the Pod.

Since requests from the Prometheus / Grafana front-end also target the NGINX server, it is able to rewrite requests to hit the appropriate endpoint on the underlying Prometheus / Grafana container.

### Default Rancher ClusterRoles

By defaults, under `templates/rancher-monitoring/clusterrole.yaml`, ClusterRoles are defined based on Rancher standards that are described in the [Rancher docs](https://ranchermanager.docs.rancher.com/v2.5/explanations/integrations-in-rancher/monitoring-and-alerting/rbac-for-monitoring#additional-monitoring-roles). These replace the default RBAC deployed for users by the upstream chart.

By default, these ClusterRoles aggregate into the [Default Kubernetes User-Facing Roles](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#user-facing-roles), which means that anyone with `admin`, `edit`, or `view` permissions (which directly corresponds to `Cluster Owner / Project Owner`, `Project Member`, and `Read-Only` Rancher roles in a given namespace) will get Prometheus Operator CR permissions on those ClusterRoles being deployed onto a cluster; this can be disabled by setting `global.cattle.userRoles.aggregateToDefaultRoles=false`.

In addition, some utility Roles are defined in `templates/rancher-monitoring/clusterrole.yaml` (specifically `monitoring-ui-view`), `templates/rancher-monitoring/config-role.yaml` / `templates/rancher-monitoring/dashboard-role.yaml` to make it easier for cluster admins to assign some minimal permissions to users to view / manage either Configmap + Secrets (i.e. Alertmanager configuration, Prometheus TLS secrets, etc.) or only ConfigMaps (Grafana dashboards / datasources). 

**Users will need to assign these roles manually** if they would like these permissions; it's not auto-aggregated into default roles by the chart.

### Leave Alertmanager Secret Behind on Uninstall

This was an intentional decision made for the Rancher Monitoring chart to only ever create the Alertmanager Secret on first install and never modify its contents of further upgrades of the chart. The purpose of this is to avoid deleting user data on chart upgrades that may have been added by modifying the Alertmanager Secret via the Rancher UI.

### Modified Default `searchNamespace` for Grafana dashboards

In order to differentiate between a user who needs the permissions to just work with Grafana Dashboard / Datasource ConfigMaps like the role in `templates/rancher-monitoring/dashboard-role.yaml` and a user who needs permissions to actually modify configuration, like the Alertmanager secret, like the role in `templates/rancher-monitoring/config-role.yaml`, the Rancher Monitoring chart needed to have different namespaces for Monitoring configuration resources (i.e. Prometheus Operator CRs and ConfigMaps that can either be attached to them or to a component like Alertmanager) v.s. the namespace used for Grafana dashboards (now `cattle-dashboards`).

As a result, the Rancher Monitoring chart deploys a namespace alongside it called `cattle-dashboards` and places all of the dashboards it deploys in that namespace. It also configures Grafana to only look for dashboards in that namespace and leaves behind the namespace on an uninstall to avoid destroying user data.

### Rancher and Nginx Ingress Grafana Dashboards

Within the `files/` directory of the Rancher Monitoring chart, some additional Grafana dashboards are added under `files/rancher/*/*`, to be rendered by the logic in `templates/rancher-monitoring/dashboards/rancher/*`.

While most of the dashboards located there are intended to parallel the same dashboards offered by upstream `kube-prometheus-stack` in `templates/grafana/dashboards-1.14`, the purpose of these dashboards are to **directly be embedded onto the Rancher Dashboard UI when Monitoring V2 is enabled in a cluster**.

The only exception to this is the Rancher performance dashboard, which is a specific dashboard that will only be deployed if users set `.Values.rancherMonitoring.enabled=true` when Monitoring V2 is deployed in the local cluster (i.e. any cluster where Rancher exists as a Deployment in the `cattle-system` namespace that has the name `rancher`).

We also copy dashboards for the NGINX Ingress Controller from upstream and place them in `files/ingress-nginx`, to be rendered by the logic in `templates/rancher-monitoring/dashboards/addons/ingress-nginx-dashboard.yaml`

### Hardening Logic

By default, under `templates/rancher-monitoring/hardened.yaml`, we deploy a set of resources that enforce that Monitoring V2 can be deployed in a **hardened** cluster, a cluster that meets [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks) by following the instructions on [Rancher's docs](https://ranchermanager.docs.rancher.com/pages-for-subheaders/rancher-v2.7-hardening-guides).

Primarily this involves:
- Deploying a default NetworkPolicy in the chart release namespace
- Running a Job that sets `automountServiceAccountToken: false` on the `default` ServiceAccount in the chart release namespace; this ensures that every workload deployed in this namespace needs a ServiceAccount to be tied to it; it cannot just use the default.


In addition, in clusters before k8s 1.25, Rancher Monitoring would also ensure that every workload deployed by the chart has a PodSecurityPolicy associated with it; however, in 1.25+, these resources have been removed from the Kubernetes API.

### Prometheus Adapter for HPA Support

Unlike upstream, Monitoring V2 enables deploying [Prometheus Adapter](https://github.com/kubernetes-sigs/prometheus-adapter) by default, which implements Kubernetes Metrics APIs (primarily the [Custom Metrics API](https://github.com/kubernetes/metrics#custom-metrics-api)) by "adapting" metrics from Rancher Monitoring Prometheus's API format over to the Kubernetes Metrics API format.

Implementing the Custom Metrics API is needed for HPA to support scaling based on Prometheus metrics because the `MetricSpec` that a `HorizontalPodAutoscaler` takes in only supports metrics that come from a Kubernetes Metrics API.

> **Note**: If a user is not using HPA, it is **safe and recommended** to shut off Prometheus Adapter.

> **Note**: You can only have **one** Prometheus adapting its metrics to the Custom Metrics or Resource Metrics API, which is why this component only exists on a Rancher Monitoring (cluster-wide) setup; deploying multiple Prometheus Adapters to adapt metrics from Project Prometheus instances are not supported.

### [`windows_exporter`](https://github.com/prometheus-community/windows_exporter) for Windows support

Unlike upstream, Monitoring V2 does not expect it is deployed on a cluster with only Linux nodes; instead, it can support RKE1 Windows clusters and RKE2 clusters that have Windows nodes as well.

Since only Windows worker nodes can be added to Kubernetes at this time, we only need to run `node-exporter`; however, node-exporter can only be used on *NIX kernels, so [`windows_exporter`](https://github.com/prometheus-community/windows_exporter) is used instead.

>**Note:** At the time that the `windows-exporter` chart was added to the `rancher-monitoring` chart, there was no repository that was packaging `windows_exporter` into an image for different Windows kernels (note: this is a special requirement for Windows servers; containers need to be built for specific kernel releases). As a result, Rancher built and maintained images under the [`rancher/windows_exporter-package`](https://github.com/rancher/windows_exporter-package) repository; these are the images that Rancher uses to package `windows_exporter` into the Rancher Monitoring chart.

>**Note:** The source of the `windows-exporter` chart exists directly in the `rancher/charts` repository as its own Package under `packages/rancher-monitoring/windows-exporter`.

> **Note**: `windows_exporter` used to be called `wmi_exporter`, since the core of the metrics it collects comes from the [Windows Management Instrumentation](https://learn.microsoft.com/en-us/windows/win32/wmisdk/wmi-start-page) running on each Windows host.

> **Note**: Technically we do need to scrape `kube-proxy` and `kubelet` as well. But since we don't need a PushProx for `kubelet` (as we directly target a ServiceMonitor at the kubelet Service and we expect the port to be available across nodes in a Kubernetes cluster based on [requiring port `10250` to be open](https://ranchermanager.docs.rancher.com/getting-started/installation-and-upgrade/installation-requirements/port-requirements#commonly-used-ports)), we don't need any Windows specific components for it. At the time of writing this document, I'm not sure if we support `kube-proxy` monitoring for Windows nodes.

### Debugging Rancher Monitoring

Most of the issues should be addressed in the [previous section on Debugging](#debugging-monitoring), but here are some additional details for specifically debugging the Rancher Monitoring pieces.

On a high level, here is the list of every **additional** component we add to Rancher Monitoring and its dependencies:
- **Windows Exporter**: No dependencies. Should be working / testable standalone.
- **PushProx Clients per all applicable Kubernetes internal component (per applicable node)**: Depends on the underlying Kubernetes internal component that it is scraping
- **PushProx Proxy per all applicable Kubernetes internal component**: Depends on PushProx clients registering with it to actually work as a proxy.
- **Prometheus Adapter**: Depends on Prometheus being healthy to be able to adapt metrics.

Here are some general tips that apply to debugging Rancher Monitoring in specialized environments that Rancher supports:
- **If you cannot run Monitoring V2 on a airgapped cluster**: check all workloads; if you are getting an `ImagePullBackoff`, it's possible that your workload either is not correctly prepending the `system_default_registry` template to that workload's image in your Helm chart to target pulling the image from your private registry (as opposed to DockerHub) or your private registry has not fully imported all the necessary images from the public DockerHub Rancher / Rancher Prime registry.
- **If you cannot run Monitoring V2 on a Windows cluster**: check that all workloads have `nodeSelectors` and `tolerations` to be deployed on the node with the correct OS; the only workload that should be deployed on Windows nodes is `windows_exporter` and all other workloads should only be on Linux nodes.
- **If you cannot run Monitoring V2 in a cluster with ARM nodes**: check that all images that have been mirrored for this chart have a manifest that supports being deployed onto a node with an ARM architecture.
- **If your workloads aren't coming up in a hardened cluster**: If workloads aren't coming up and this is a 1.24 or less Kubernetes cluster, check to see that the workload is attached to a `ServiceAccount` that is referenced by a `ClusterRoleBinding` / `RoleBinding` that attaches it to a `ClusterRole` / `Role` that allows it to use a `PodSecurityPolicy` that provides the necessary permissions for your workload to deploy; there should be a more detailed error message in the Kubernetes events for this object if this is the issue that indicates the missing permission. If workloads aren't coming and this is a 1.25+ cluster, check to see that the `cattle-monitoring-system` namespace has the appropriate [`Pod Security Admission`](https://kubernetes.io/docs/concepts/security/pod-security-admission/) label to deploy `privileged` containers **(required for PushProx clients to be deployed)** or that the `cattle-monitoring-system` namespace exists in the list of [Exempted Namespaces for Pod Security Admission](https://kubernetes.io/docs/concepts/security/pod-security-admission/#exemptions). If a CIS scan is failing, check that the `cattle-monitoring-system` and `cattle-dashboards` namespace contain a `ServiceAccount` named `default` that has `automountServiceAccountToken` set to false. Also, check that those namespaces also contain default `NetworkPolicies`.
- **If your workloads aren't coming up in a cluster with [`seLinux`](https://www.redhat.com/en/topics/linux/what-is-selinux) enabled on the nodes**: ensure `global.seLinux.enabled=true` when deploying the chart, which triggers providing `seLinuxOptions` to the `securityContext` for all relevant components that need it (i.e. all components that mount `hostPaths`, such `rkeEtcd`, where SELinux permissions are required). If a particular component does not have `seLinuxOptions` but needs it because it uses a hostPath, **each** container in the workload that uses the hostPath will need to set securityContext with those `seLinuxOptions`. Generally, this will involve setting `securityContext.seLinuxOptions.type` equal to a new or existing type tracked in [`rancher/rancher-selinux`](https://github.com/rancher/rancher-selinux) that represents what your process is trying to do (e.g. Monitoring V2 creates a `rke_kubereader_t` type to represent a type of process that needs to be able to access all Kubernetes files on the node). If you needed to add a type to [`rancher/rancher-selinux`](https://github.com/rancher/rancher-selinux), you'll need to package your copy into an RPM (or cut a tag in the repository to create the RPM) and install that RPM to each node in the cluster before testing your changes.

Here are some tips that apply to debugging Monitoring's integration with Rancher when deployed onto a Rancher setup:
- **If a Rancher User cannot see a Monitoring UI**: The permissions to access the Rancher UI are based on what requests are being made in [within the custom UI component in `rancher/dashboard`](https://github.com/rancher/dashboard/tree/master/shell/chart/monitoring); if there's a regression in the user expectations, it's probably a Rancher UI issue. You can verify that this is the case by modifying your Rancher `Global Settings` for `ui-dashboard-index` to point at an older Rancher dashboard version and updating `ui-offline-preferred` to `Remote` and refreshing your page; if you no longer see the issue, it's probably a UI regression. Currently, as listed in [this closed issue](https://github.com/rancher/dashboard/issues/2408), the expectations are that: 1) having `service/proxy` permissions to the Monitoring components will allow you to access the UIs if someone gives you a link, but it won't show up on the Rancher UI 2) having `get` permissions for the `PodMonitor` custom resource in any namespace will allow you to see the Monitoring UI pane with the links, but won't actually allow you to click on them and 3) having permissions to list `Endpoints` in the `cattle-monitoring-system` namespace and `get` all Workload types will allow the UI to actually make to call to ensure that the components are up and healthy to allow you to click on the links.
- **If you aren't seeing any Grafana dashboards embedded on Rancher's Dashboard UI**: See notes above regarding the possibility of a UI regression. You should also ensure that the dashboard exist on Grafana; if not, you should follow the steps for Grafana in the [Debugging Monitoring docs](#debugging-monitoring).
- **If a Rancher User create, modify, or delete a Prometheus Operator resource**: Please refer to the section on `Prometheus Operator` with respect to the admission webhook in the [Debugging Monitoring docs](#debugging-monitoring). It's also possible you're running into an issue related to Rancher interfering with Prometheus Operator custom resource as detailed in the [CRD chart section](#crd-chart-with-install--uninstall-jobs).

Here are some component-specific tips for debugging Rancher Monitoring:
- **If your Component is a non-PushProx Exporter (`windows_exporter`)**: refer to the section above in [Debugging Monitoring](#debugging-monitoring) for Exporters.
- **If your Component is any part of a PushProx Exporter**: first ensure that the underlying component you are scraping is healthy by following the instructions for Exporters on [Debugging Monitoring](#debugging-monitoring) (execute into the PushProx client pod or, if unavailable, the node itself instead of the Prometheus container). For some exporters like `rkeEtcd`, you may need to run the same `wget` / `curl` command to the metrics endpoint using hostPath certificates or `insecureSkipVerify`; see the PushProx configuration on the `values.yaml` of the `rancher-monitoring` chart for more information on what is expected. Then ensure the workloads themselves for the PushProx clients and proxy are as expected; that they are healthy and are deployed onto the expected nodes. If that looks fine, check to see if the PushProx clients are able to establish a connnection with the PushProx proxy (if this wasn't the case, the logs would probably indicate this though). Then check to see if Prometheus can establish a connection with the PushProx proxy. Finally, if all else fails, run an HTTP request using wget from the Prometheus Pod after setting the `http_proxy` environment variable to point to the PushProx proxy pod and ensure that you can fully execute a scrape request.
- **If your Component is Prometheus Adapter**: refer to the section above in [Debugging Monitoring](#debugging-monitoring) for Prometheus. If Prometheus is healthy, check the configuration of the Prometheus Adapter secret to see if it might be malformed; if it is not, check to see if metrics are being exported correctly to the Custom Metrics or External Metrics API.

## Add-On Chart: Alerting Drivers

The Alerting Drivers chart is an **independently deployable** "add-on" chart to the Rancher Monitoring chart; when deployed into the cluster, it allows Rancher Monitoring to proxy requests to the deployed drivers to support sending notifications to non-native receivers like [SMS via `messagebird/sachet`](https://github.com/messagebird/sachet) or [Microsoft Teams via `idealista/prom2teams`](https://github.com/idealista/prom2teams).

### What is a Non-Native Receiver in Alertmanager?

As listed in the [Alertmanager docs](https://prometheus.io/docs/operating/integrations/#alertmanager-webhook-receiver), there are a lot of non-native receivers that support receiving notifications from Alertmanager after you have configured Alertmanager with a [`webhook_config-based receiver`](https://prometheus.io/docs/alerting/latest/configuration/#webhook_config).

Essentially, on parsing an alert through Alertmanager's routing tree and finding that it needs to go to a specific Webhook Receiver, Alertmanager will send an HTTP request using the provided `http_config` to whatever `url` the receiver is pointing to (which generally is some Kubernetes DNS pointing to the receiver, like `http://rancher-sachet.cattle-monitoring-system.svc`) and expect that the receiver will handle forwarding that request to the desired notification mechanism (Microsoft Teams, SMS, etc.).

### How Alerting Drivers Works

Alerting Drivers allows you to deploy one or more of:
- [SMS via `messagebird/sachet`](https://github.com/messagebird/sachet)
- [Microsoft Teams via `idealista/prom2teams`](https://github.com/idealista/prom2teams)

After deploying the driver(s), you will then have to follow the configuration options of the upstream driver to configure the notification being sent to your desired location.

Once the drivers are configured, you can use the Rancher UI to set up a Receiver pointing to the drivers (the Rancher UI will automatically infer the expected URL to send the notification to based on the fact that Alerting Drivers is installed), and start sending alerts to your configured notifier.

### Debugging Alerting Drivers

Since Alerting Drivers depends on Monitoring's functionality, you should first ensure that your Monitoring stack is fully functional and is capable of sending alerts to a normal receiver.

Once that has been confirmed, you will want to deploy Alerting Drivers and ensure that the logs don't show any errors; then you should add some default configuration to the relevant Driver's configuration and simulate Alertmanager by directly making an HTTP call (similar to the [script provided below for Alertmanager](#example-send-alerts-to-alertmanager)) to send an alert to your driver.

If the driver does not work, there's probably something wrong with your configuration. If the driver works without Alertmanager, then try adding the receiver to your Alertmanager configuration and ensure you are able to receive your alerts as expected.

If you don't receive alerts, its possible that the way that Alertmanager is translating what alert to send to your driver may be incorrect. To debug that, try setting up a simple server that echos the HTTP requests it receives ([something like this](https://code.mendhak.com/docker-http-https-echo/)) and target the `url` of the `webhook_config` on Alertmanager to point to that instance instead; by seeing the difference between the alert that Alertmanager sends and what you tried to send in the previous step, you should be able to identify where the issue might be coming from.

## Additionals

### Example: Send Alerts To Alertmanager

```bash
#!/bin/bash

# A link to Alertmanager
#
# Must be accessible to the network where you are running this curl
# command; for example, you can't run this on your local laptop unless
# it's part of a Kubernetes cluster's network (i.e. using kubectl proxy)
#
# For example, you could run this in a pod in your cluster to avoid
# needing to authenticate with the Kubernetes API by directly pinging
# Alertmanager via Kubernetes DNS
#
ALERTMANAGER_UI=

curl -XPOST ${ALERTMANAGER_UI}/api/v1/alerts -H "Accept: application/json" -d '[{ 
	"status: "firing",
	"labels: {
		"alertname": "$name",
		"service": "my-service",
		"severity":"warning",
		"instance": "$name.example.net",
        "extraLabel": "arvind",
        "job": "test-alert-$RANDOM",
        "cluster": "mytestcluster"
	},
	"annotations": {
		"summary": "MY SUMMARY HERE",
        "description": "my description here",
        "message": "My MeSsAgE hErE",
        "extraAnnotation": "rancher rocks"
	},
	"generatorURL": "http://prometheus.int.example.net/<generating_expression>"
}]'
```

> **Note**: If you are sending alerts via Alertmanager deployed via Rancher, you can copy your Rancher `service/proxy` link to Alertmanager as `${ALERTMANAGER_UI}` but you will also need to provide an auth token (i.e. `-H "Authorization: Bearer ${TOKEN}"`) to authenticate you to the Rancher API.

### Links

For an architecture overview, please read the Rancher docs on [How Monitoring Works](https://ranchermanager.docs.rancher.com/v2.6/integrations-in-rancher/monitoring-and-alerting/how-monitoring-works).

