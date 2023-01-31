Related Issue: <!-- link the issue or issues this PR resolves here -->
<!-- If your PR depends on changes from another pr link them here and describe why they are needed on your solution section. -->

### Checklist

Please fill out this table to identify which fields need to be modified in your PR.

**Under `Status`, either indicate `Does Not Apply` or `Added to this PR`**.

| **Version to be incremented**                                              | Why should this be modified?                                                                                                           | Status                                      |
|-----------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------|
| `version` in rancher-project-monitoring `Chart.yaml`                  | You modified the contents of the `rancher-project-monitoring` chart to make changes                                                    |  |
| `helmProjectOperator.image.tag` in prometheus-federator `values.yaml` | Either you modified the rancher-project-monitoring chart or you modified the `main.go` file                                          |  |
| `appVersion` in prometheus-federator `Chart.yaml`                     | You modified the `helmProjectOperator.image.tag` in the above box                                                                      |  |
| `version` in prometheus-federator `Chart.yaml`                        | Either you modified the `appVersion` in the above box or you modified the contents of the `prometheus-federator` chart to make changes |  |

