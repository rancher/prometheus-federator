helm-locker
========

Helm Locker is a Kubernetes operator that prevents resource drift on (i.e. "locks") Kubernetes objects that are tracked by Helm 3 releases.

Once installed, a user can create a `HelmRelease` CR in the `Helm Release Registration Namespace` (default: `cattle-helm-system`) by providing:
1. The name of a Helm 3 release
2. The namespace that contains the Helm Release Secret (supplied as `--namespace` on the `helm install` command that created the release)

Once created, the Helm Locker controllers will watch all resources tracked by the Helm Release Secret and automatically revert any changes to the persisted resources that were not made through Helm (e.g. changes that were directly applied via `kubectl` or other controllers).

## Who needs Helm Locker?

Anyone who would like to declaratively manage resources deployed by existing Helm chart releases.

## How is this different from projects like `fluxcd/helm-controller`?

Projects like [`fluxcd/helm-controller`](https://github.com/fluxcd/helm-controller) allow users to declaratively manage **Helm charts from deployment to release**, whereas this project only allows you lock an **existing** Helm chart release; as a result, the scope of this project is much more narrow than what is offered by `fluxcd/helm-controller` and should be integrable with any solution that produces Helm releases.

If you are looking for a larger, more opinionated solution that also has features around **how** Helm charts should be deployed onto a cluster (e.g. from a `GitRepository` or `Bucket` or `HelmRepository`), this is not the project for you.

However, if you are looking for something light-weight that simply guarentees that **Helm is the only way to modify resources tracked by Helm releases**, this is a good solution to use.

## How does Helm Locker know whether a release was changed by Helm or by another source?

In order to prevent multiple Helm instances from performing the same upgrade at the same time, Helm 3 will always first update the `info.status` field on a Helm Release Secret from `deployed` to another state (e.g. `pending-upgrade`, `pending-install`, `uninstalling`, etc.) before performing the operation; once the operation is complete, the Helm Release Secret is expected to be reverted back to `deployed`.

Therefore, if Helm Locker observes a Helm Release Secret tied to a `HelmRelease` has been updated, it will check to see what the current status of the release is; if the release is anything but `deployed`, Helm Locker will not perform any operations on the resources tracked by this release, which will allow upgrades to occur as expected. 

However, once a release is `deployed`, if what is tracked in the Helm secret is different than what is currently installed onto the cluster, Helm Locker will revert all resources back to what was tracked by the Helm release (in case a change was made to the resource tracked by the Helm Release while the release was being modified).

## Debugging

### How do I manually inspect the content of the Helm Release Secret to debug a possible Helm Locker issue?

Identify the release namespace (`RELEASE_NAMESPACE`), release name (`RELEASE_NAME`), and release version (`RELEASE_VERSION`) that identifies the Secret used by Helm to store the release data. Then, with access to your Kubernetes cluster via `kubectl`, run the following command (e.g. run base64 decode, base64 decode, gzip decompress the .data.release of the Secret):

```bash
RELEASE_NAMESPACE=default
RELEASE_NAME=test
RELEASE_VERSION=v1

# Magic one-liner! jq call is optional...
kubectl get secrets -n ${RELEASE_NAMESPACE} sh.helm.release.v1.${RELEASE_NAME}.${RELEASE_VERSION} -o=jsonpath='{ .data.release }' | base64 -d | base64 -d | gunzip -c | jq -r '.'
```

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
