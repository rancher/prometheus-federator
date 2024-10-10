prometheus-federator
========

This repo contains a set of three interlinked projects:

- The **Prometheus Federator** is a k8s Operator that manages deploying Project Monitoring Stacks.
- The **Helm Project Operator** is a generic design for a Kubernetes Operator that acts on `ProjectHelmChart` CRs.
- **Helm Locker** is a Kubernetes operator that prevents resource drift on (i.e. "locks") Kubernetes objects that are tracked by Helm 3 releases.

> [!NOTE]
> The last two project (helm-project-operator and helm-locker) are not intended or supported for standalone use.

For more info on _Helm Project Operator_, see the [dedicated README file](README-helm-project-operator.md).  
For more info on _Helm Locker_, see the [dedicated README file](README-helm-locker.md).

## Getting Started

For more information, see the [Getting Started guide](docs/prometheus-federator/gettingstarted.md).

## More Info

Prometheus Federator is an operator (powered by [`rancher/helm-project-operator`](README-helm-project-operator.md) and [`rancher/charts-build-scripts`](README-helm-locker.md)) that manages deploying one or more Project Monitoring Stacks composed of the following set of resources that are scoped to project namespaces:
- [Prometheus](https://prometheus.io/) (managed externally by [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator))
- [Alertmanager](https://prometheus.io/docs/alerting/latest/alertmanager/) (managed externally by [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator))
- [Grafana](https://github.com/helm/charts/tree/master/stable/grafana) (deployed via an embedded Helm chart)
- Default PrometheusRules and Grafana dashboards based on the collection of community-curated resources from [kube-prometheus](https://github.com/prometheus-operator/kube-prometheus/)
- Default ServiceMonitors that watch the deployed Prometheus, Grafana, and Alertmanager

A user can specify that they would like to deploy a Project Monitoring Stack by creating a `ProjectHelmChart` CR in a Project Registration Namespace (`cattle-project-<id>`) with `spec.helmApiVersion: monitoring.cattle.io/v1alpha1`, which will deploy the Project Monitoring Stack in a Project Release Namespace (`cattle-project-<id>-monitoring`). 

> Note: Since this Project Monitoring Stack deploys Prometheus Operator CRs, an existing Prometheus Operator instance must already be deployed in the cluster for Prometheus Federator to successfully be able to deploy Project Monitoring Stacks. It is recommended to use [`rancher-monitoring`](https://rancher.com/docs/rancher/v2.6/en/monitoring-alerting/) for this. For more information on how the chart works or advanced configurations, please read the [`README.md` on the chart](packages/prometheus-federator/charts/README.md).

For more information on ProjectHelmCharts and how to configure the underlying operator, please read the [`README.md` on the chart](packages/prometheus-federator/charts/README.md) or check out the general docs on Helm Project Operators in [`rancher/helm-project-operator`](https://github.com/rancher/helm-project-operator).

For more information on how to configure the underlying Project Monitoring Stack, please read the [`README.md` of the underlying chart](packages/rancher-project-monitoring/charts/README.md) (`rancher-project-monitoring`).

## Developing

### Which branch do I make changes on?

Prometheus Federator is built and released off the contents of the `main` branch. To make a contribution, open up a PR to the `main` branch.

For more information, see the [Developing guide](docs/prometheus-federator/developing.md).

## Building

`make`

> **Note:** For a more in-depth explanation of how Prometheus Federator is built (intended for anyone who would like to fork this repo to create a new Project Operator!), see the [Build guide](docs/prometheus-federator/build.md).

## Running

`./build/bin/prometheus-federator`

## Versioning and Releasing For Rancher

While this repository does maintain a standalone Helm repository for vanilla Helm users to consume directly, users of Rancher will see forked versions of these chart releases available on Rancher's Apps & Marketplace; the forked chart releases are maintained in the [`rancher/charts`](https://github.com/rancher/charts) repository on being released from a `dev-vX.X` branch to a `release-vX.X` branch, where `X.X` corresponds to the Rancher `${Major}.${Minor}` version that the users is using (i.e. Rancher `2.7`). 

**The chart in rancher/charts is generally the version that is intended for use in production since that is the chart that will be tested by Rancher's QA team.** Generally, these charts will match stable versions of charts available in this repository, so non-Rancher users **should** be able to safely use those versions in this repository for production use cases (at their own risk).

For more information on the process maintainers of this repository use to mirror these charts over to [`rancher/charts`](https://github.com/rancher/charts), see the [Rancher release guide](docs/prometheus-federator/rancher_release.md).

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
