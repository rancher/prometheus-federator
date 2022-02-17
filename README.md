prometheus-federator
========

The Prometheus Federator is intended to be deployed in a Kubernetes cluster running an instance of [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator) and a cluster-wide instance of a [Prometheus](https://prometheus.io) CR deployed through [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack).

The primary purpose of this operator is to allow users to create `Projects`, groups of namespaces (selected via label selectors) that should be tracked and monitored by independent instances of Prometheus, Alertmanager, and Grafana.

Instead of having each Prometheus independently scrape a set of exporters, each Project Prometheus utilizes [federation](https://prometheus.io/docs/prometheus/latest/federation/) to scrape a pre-configured and pre-existing Cluster Prometheus that will be responsible for collecting metrics from the following exporters:
- [node_exporter](https://github.com/prometheus/node_exporter)
- [windows_exporter](https://github.com/prometheus-community/windows_exporter)
- [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics)

On starting Prometheus Federator, users will need to provide the name and namespace containing the Cluster Prometheus CR that will serve as the cluster-level aggregator of metrics.

In addition, Prometheus Federator is expected to be deployed alongside a Federator PrometheusRule CR, which will create a set of default recording rules on the cluster Prometheus to group together metrics by namespaces in the cluster. This will be packaged in the Helm chart used to deploy the Prometheus Federator.

On initialization, Prometheus Federator will watch the designated Cluster Prometheus CR that will serve as the cluster-level aggregator of metrics; it will identify any namespaces that are selected by the Cluster Prometheus CR and automatically prevent `Projects` from selecting any namespaces that are already targeted by the Cluster Prometheus (note: if a `Project` cannot target any namespaces as a result, a status will be updated on the resource to indicate this. `Projects` will also be limited from selecting other `Project` namespaces by default).

Once it is up and running, users can define `Projects` in the project registration namespace, which by default will be the namespace that Prometheus Federator is deployed within.

When a Project is created, Prometheus Federator will automatically create and manage the following resources per CR:
- A Project Namespace, created to host resources for a given project
- A Project Prometheus CR, which will be configured to [federate](https://prometheus.io/docs/prometheus/latest/federation) namespace-scoped metrics generated from the Federator PrometheusRule on the Cluster Prometheus via a PodMonitor. A PrometheusRule CR will also be created in the Project namespace that will aggregate these namespace-scoped metrics into project-scoped metrics via recording rules and set up alerting rules to send out alerts.
- A Project Alertmanager CR (optional, defined in the Project CR) that the Prometheus CR will be configured to send alerts to
- A Deployment of Project Grafana, which will pull data from Prometheus to generate Grafana dashboards visualizing project-scoped and namespace-scoped metrics

## Building

`make`


## Running

`./bin/prometheus-federator`

## License
Copyright (c) 2020 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
