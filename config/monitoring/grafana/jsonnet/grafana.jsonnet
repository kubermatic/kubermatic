local k = import 'ksonnet/ksonnet.beta.3/k.libsonnet';

local datasources = [
  {
    name: "prometheus",
    type: "prometheus",
    access: "proxy",
    org_id: 1,
    url: "http://prometheus-kubermatic.monitoring.svc.cluster.local:9090",
    version: 1,
    editable: false,
    default: true,
  },
];

local g =
(import 'grafana/grafana.libsonnet') +
(import 'kubernetes-mixin/mixin.libsonnet') +
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

    versions+:: {
      grafana: '{{ .Values.grafana.image.tag }}',
    },

    imageRepos+:: {
      grafana: '{{ .Values.grafana.image.repository }}',
    },

    grafana+:: {
      dashboards: $.grafanaDashboards,
      datasources: datasources,
      config: {}, // This will add the config reference to the deployment, but we're using our own with helm
    },
  },
};

// Create a new object to have a list with all dashboards.
{
  dashboardDatasources: g.grafana.dashboardDatasources,
  dashboardDefinitions: k.core.v1.list.new(g.grafana.dashboardDefinitions),
  dashboardSources: g.grafana.dashboardSources,
  deployment: g.grafana.deployment,
  serviceAccount: g.grafana.serviceAccount,
  service: g.grafana.service,
}
