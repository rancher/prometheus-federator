helm-project-operator
========

This repo contains a set of two interlinked projects:

- The **Helm Project Operator** is a generic design for a Kubernetes Operator that acts on `ProjectHelmChart` CRs.
- **Helm Locker** is a Kubernetes operator that prevents resource drift on (i.e. "locks") Kubernetes objects that are tracked by Helm 3 releases.

**Note: These project are not intended for standalone use.** 

For more info on _Helm Locker_, see the [dedicated README file](README-helm-locker.md).

Helm Project Operator is intended to be implemented by a Project Operator (e.g. [`rancher/prometheus-federator`](https://github.com/rancher/prometheus-federator)) but provides a common definition for all Project Operators to use in order to support deploy specific, pre-bundled Helm charts (tied to a unique registered `spec.helmApiVersion` associated with the operator) across all project namespaces detected by this operator.

## Getting Started

For more information, see the [Getting Started guide](docs/helm-project-operator/gettingstarted.md).

## Developing

### Which branch do I make changes on?

Helm Project Operator is built and released off the contents of the `main` branch. To make a contribution, open up a PR to the `main` branch.

For more information, see the [Developing guide](docs/helm-project-operator/developing.md).

## Design

Helm Project Operator is built on top of [k3s-io/helm-controller](https://github.com/k3s-io/helm-controller) and [rancher/helm-locker](https://github.com/rancher/helm-locker). For more information on the design of the underlying components, please see the `README.md` on their respective repositories.

For an example of how Helm Project Operator can be implemented, please see [`rancher/prometheus-federator`](https://github.com/rancher/prometheus-federator).

For more information in general, please see [docs/design.md](docs/helm-project-operator/design.md).

## Building

`make`

## Running

`./bin/helm-project-operator`

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
