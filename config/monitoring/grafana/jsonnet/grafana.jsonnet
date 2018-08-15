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
      config: {}, // This will add the config reference to the deployment, but we're using our own with helm
    },
  },
};

g.grafanaDashboards