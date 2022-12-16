Related Issue: <!-- link the issue or issues this PR resolves here -->
<!-- If your PR depends on changes from another pr link them here and describe why they are needed on your solution section. -->

### Checklist

In general, the rules for bumping versions are:
- If the current version is an RC, increment it `0.0.0-rc1` -> `0.0.0-rc2`
- If the current version is not an RC, bump the patch and set it to `-rc1` (`0.0.0` -> `0.0.1-rc1`)

Please fill out this table to identify which fields need to be modified in your PR.

| **Field to be modified**                                              | Why should this be modified?                                                                                                           | Status                                      |
|-----------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------|
| `version` in rancher-project-monitoring `Chart.yaml`                  | - You modified the contents of the rancher-project-monitoring chart to make changes                                                    | **Does Not Apply** / **Added to this PR**   |
| `helmProjectOperator.image.tag` in prometheus-federator `values.yaml` | Either: - You modified the rancher-project-monitoring chart - You modified the `main.go` file                                          | **Does Not Apply**  /  **Added to this PR** |
| `appVersion` in prometheus-federator `Chart.yaml`                     | You modified the `helmProjectOperator.image.tag` in the above box                                                                      | **Does Not Apply**  /  **Added to this PR** |
| `version` in prometheus-federator `Chart.yaml`                        | Either: - You modified the `appVersion` in the above box - You modified the contents of the prometheus-federator chart to make changes | **Does Not Apply**  /  **Added to this PR** |

### For Maintainers Releasing The Chart On QA Validation

Please checkmark **both** of the boxes below to indicate that you have followed the versioning guidelines for `prometheus-federator`:
- [ ] The `-rc` tag has been removed from the `version` in `packages/prometheus-federator/charts/Chart.yaml`
- [ ] The `-rc` tag has been removed from the `helmProjectOperator.image.tag` in `packages/prometheus-federator/charts/values.yaml`

