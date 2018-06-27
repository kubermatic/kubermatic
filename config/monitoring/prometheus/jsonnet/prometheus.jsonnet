local k = import 'ksonnet/ksonnet.beta.3/k.libsonnet';

(import 'kubernetes-mixin/mixin.libsonnet') + {
  _config+:: {
    namespace: 'monitoring',

    jobs+:: {
        KubeAPI: 'job="apiserver"',
    },

    prometheus+:: {
      name: 'testing',
      rules: $.prometheusRules + $.prometheusAlerts,
    },
  },

  prometheus+:: {
    rules+:
      $._config.prometheus.rules,
  },
}
