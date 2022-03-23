# Migrate to prometheus-operator for master/seed MLA

**Author**: Wojciech Urbański (@wurbanski)

**Status**: Draft proposal

## Motivation and Background

[Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator) can be used to manage monitoring stack within KKP platform instead of deploying specific components and managing them using our internal controllers. With this approach most of the configuration we are doing manually right now could be moved into a [set of CRDs defined by the operator](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/api.md).

1. During the work on migration of the charts, it was identified that the `monitoring/prometheus` chart is integrated tightly 
with the other components, e.g. by embedding scraping and alerting rules within the main chart, triggered by specific entries in values.yaml. 
Hence, while making changes to any of other components, it is also required to modify the `prometheus` chart to include specific rules.

    This is not a problem in the informed development process, but it's not a common pattern - most of publicly 
available charts contain required configuration for Prometheus stack within, which makes it easy to edit and manage the rules related to the specific product.

    Separating the "management" (operator) and "data" (configs per services) also makes it much easier to upgrade the components (and their charts) separately.

2. In current state, seed-controller-manager managed the installations of Prometheus' instances in cluster namespaces on the seed cluster. This can lead to unwanted discrepancy between the versions used for different instances, as main KKP seed Prometheus instance is installed using a Helm chart.

    This behaviour could be rectified by using the prometheus-operator to govern the installation and management of all instances of Prometheus used within the seed.

3. One of the reasons for not adopting the most popular upstream monitoring suite - [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack) is a lot of noise due to many default rules, but the rulesets can be customized using [values.yaml](https://github.com/prometheus-community/helm-charts/blob/main/charts/kube-prometheus-stack/values.yaml#L33-L59) entries.

    KKP-internal configuration of Prometheus instances can be done using the CRDs instead of configmaps/secrets.


Additional note: almost 4 years ago [a decision was made](https://github.com/kubermatic/kubermatic/issues/947) to move away from prometheus-operator in KKP, however the operator has matured a lot since that time and has now become a de-facto standard for managing Prometheus installations. Switching to the operator brings additional benefit of just being easier to comprehend for new admins/developers already used to the manner of configuration introduced by prometheus-operator.

## Implementation

Implementation consists of multiple parts:

1. Mapping existing configuration of Prometheus and monitoring stack to CRs and configuration options of upstream kube-prometheus-stack.
   1. moving per-service configuration to charts providing the service (e.g. Velero-related alerts are moved to the Velero chart), replacing annotations-based config with ServiceMonitors.
   2. rewriting scraping rules to `PrometheusRule` CRs
2. Extract Thanos to separate, additional chart OR use [bitnami/thanos](https://artifacthub.io/packages/helm/bitnami/thanos)
3. Rewriting resource management logic in seed-controller-manager to manage the operator's custom resources, e.g. `Prometheus` instead of `StatefulSet`, `PrometheusRule` instead of some `ConfigMap`s).
   1. Manage `Prometheus` CRs instead of StatefulSets
   2. Manage `PrometheusRule`s instead of `ConfigMaps`
   3. Use a single `PodMonitor` object targeting each other Prometheus instance in all `cluster-xxxxxx` namespaces. Configuration of federation would be handled once.
   4. Each of the per-usercluster Prometheus instance has to be configured with specific etcd secrets used to scrape etcd-server metrics.

4. Define plan for migrating the data between old and new Prometheus instances.
   1. Technically, the data can be copied over from one instance to another and PVCs should be saved between the old and new instances. A manual for the required steps should be enough.


## Task & effort:

TBD
