local g =
(import 'grafana/grafana.libsonnet') +
(import 'kubernetes-mixin/mixin.libsonnet') +
(import 'etcd-mixin/mixin.libsonnet') +
(import './dashboards/kubermatic/kubermatic.libsonnet') +
{ _config+:: {
    namespace: 'monitoring',

    cadvisorSelector: 'job="cadvisor"',
    kubeletSelector: 'job="kubelet"',
    kubeStateMetricsSelector: 'job="kube-state-metrics"',
    nodeExporterSelector: 'app="node-exporter"',
    notKubeDnsSelector: 'job!="kube-dns"',
    kubeSchedulerSelector: 'job="kube-scheduler"',
    kubeControllerManagerSelector: 'job="kube-controller-manager"',
    kubeApiserverSelector: 'job="apiserver"',
    machineControllerSelector: 'job="machine-controller"',
  },
};

{
  "kubernetes/nodes.json": g.grafanaDashboards["nodes.json"],
  "kubernetes/pods.json": g.grafanaDashboards["pods.json"],
  "kubernetes/resources-cluster.json": g.grafanaDashboards["k8s-resources-cluster.json"],
  "kubernetes/resources-namespace.json": g.grafanaDashboards["k8s-resources-namespace.json"],
  "kubernetes/resources-pod.json": g.grafanaDashboards["k8s-resources-pod.json"],
  "kubernetes/statefulset.json": g.grafanaDashboards["statefulset.json"],
  "kubernetes/cluster-rsrc-use.json": g.grafanaDashboards["k8s-cluster-rsrc-use.json"],
  "kubernetes/node-rsrc-use.json": g.grafanaDashboards["k8s-node-rsrc-use.json"],
  "kubernetes/etcd.json": g.grafanaDashboards["etcd.json"],

  "kubermatic/nginx.json": g.grafanaDashboards["nginx.json"],
  "kubermatic/machine-controller.json": g.grafanaDashboards["machine-controller.json"],
}
