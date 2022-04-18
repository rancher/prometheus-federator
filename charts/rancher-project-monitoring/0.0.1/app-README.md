# Rancher Project Monitoring and Alerting

The chart installs the following components:

- [Project Prometheus]() - Prometheus is a time series database that collects metrics federated from a Cluster Prometheus
- [Project Alertmanager]() - Alertmanager allows a user to send alerts to configured notification providers
- [Project Grafana](https://github.com/helm/charts/tree/master/stable/grafana) - Grafana allows a user to create / view dashboards based on the cluster metrics collected by Prometheus.
- [A subset of kube-prometheus](https://github.com/prometheus-operator/kube-prometheus/) - A collection of community-curated Kubernetes manifests, Grafana Dashboards, and PrometheusRules that deploy a default end-to-end cluster monitoring configuration.


