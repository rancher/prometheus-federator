## Issue: <!-- link the issue or issues this PR resolves here -->
<!-- If your PR depends on changes from another pr link them here and describe why they are needed on your solution section. -->
 
## Problem
<!-- Describe the root cause of the issue you are resolving. This may include what behavior is observed and why it is not desirable. If this is a new feature describe why we need this feature and how it will be used. -->
 
## Solution
<!-- Describe what you changed to fix the issue. Relate your changes back to the original issue / feature and explain why this addresses the issue. -->
 
## Testing
<!-- Note: Confirm if the repro steps in the GitHub issue are valid, if not, please update the issue with accurate repro steps. -->

## Versioning

### For Community Members or Maintainers Making Changes

Please checkmark one of the boxes below to indicate you have following the versioning guidelines for `rancher-project-monitoring`:

- If you are introducing a change to `packages/rancher-project-monitoring` or `packages/rancher-project-grafana`:
  - [ ] Increment the patch version in the `version` of `packages/rancher-project-monitoring/charts/Chart.yaml` by 1
- [ ] I am not introducing a change to `package/rancher-project-monitoring`

> **Note:** We do not use RC versions for `rancher-project-monitoring` since it is hidden anyways and not intended for standalone use

Please checkmark one of the boxes below to indicate that you have followed the versioning guidelines for `prometheus-federator`:

- If you are introducing a change to `main.go` or `packages/rancher-project-monitoring` (including a change introduced in the above step):
  - [ ] If `packages/prometheus-federator/charts/Chart.yaml` has a `version` that is a `-rc` version, increment the `-rc` version in this file by one (i.e. `0.1.2-rc1` -> `0.1.2-rc2`). Modify the `appVersion` to match this new `version`. Modify the `helmProjectOperator.image.tag` in `packages/prometheus-federator/charts/values.yaml` to match this `appVersion`.
  - [ ] If `packages/prometheus-federator/charts/Chart.yaml` has a `version` that is **not** a `-rc` version, increment the patch version in this file by 1 and add `-rc1` (i.e. `0.1.1` -> `0.1.2-rc1`). Modify the `appVersion` to match this new `version`. Modify the `helmProjectOperator.image.tag` in `packages/prometheus-federator/charts/values.yaml` to match this `appVersion`.
- If you are **only** introducing a change to `packages/prometheus-federator`:
  - [ ] If `packages/prometheus-federator/charts/Chart.yaml` has a `version` that is a `-rc` version, increment the `-rc` version in this file by one (i.e. `0.1.2-rc1` -> `0.1.2-rc2`). **Do not modify the `appVersion` or the `helmProjectOperator.image.tag` in `packages/prometheus-federator/charts/values.yaml`**.
  - [ ] If `packages/prometheus-federator/charts/Chart.yaml` has a `version` that is **not** a `-rc` version, increment the patch version by 1 in this file and add `-rc1` (i.e. `0.1.1` -> `0.1.2-rc1`). **Do not modify the `appVersion` or the `helmProjectOperator.image.tag` in `packages/prometheus-federator/charts/values.yaml`**.

### For Maintainers Releasing The Chart On QA Validation

Please checkmark **both** of the boxes below to indicate that you have followed the versioning guidelines for `prometheus-federator`:
- [ ] The `-rc` tag has been removed from the `version` in `packages/prometheus-federator/charts/Chart.yaml`
- [ ] The `-rc` tag has been removed from the `helmProjectOperator.image.tag` in `packages/prometheus-federator/charts/values.yaml`

## Engineering Testing
### Manual Testing
<!-- Describe what manual testing you did (if no testing was done, explain why). -->

### Automated Testing
<!--If you added/updated unit/integration/validation tests, describe what cases they cover and do not cover. -->

## QA Testing Considerations
<!-- Highlight areas or (additional) cases that QA should test w.r.t a fresh install as well as the upgrade scenarios -->
 
### Regressions Considerations
<!-- Dedicated section to specifically call out any areas that with higher chance of regressions caused by this change, include estimation of probability of regressions -->

