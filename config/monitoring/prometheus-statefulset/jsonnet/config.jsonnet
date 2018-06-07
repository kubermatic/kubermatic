{
  _config+:: {
    namespace: 'monitoring',
  },

  config+: {
    global+: {
      scrape_interval: '15s',
      evaluation_interval: '15s',
    },

    rule_files+: ['/etc/prometheus/rules/*.yaml'],

    alerting+: {
      alertmanagers+: [
        {
          kubernetes_sd_configs+: [
            {
              api_server: null,
              role: 'endpoints',
              namespaces: {
                names: [
                  $._config.namespace,
                ],
              },
            },
          ],
        },
      ],
    },
  },
}.config
