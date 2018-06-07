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

    // alerting+: {
    //   alertmanagers+: [
    //     {
    //       kubernetes_sd_configs+: [
    //         {
    //           api_server: null,
    //           role: 'endpoints',
    //           namespaces: {
    //             names: [
    //               $._config.namespace,
    //             ],
    //           },
    //         },
    //       ],
    //     },
    //   ],
    // },

    scrape_configs+: [
      {
        job_name: 'kubernetes-apiservers',
        kubernetes_sd_configs: [{ role: 'endpoints' }],
        tls_config: {
          ca_file: '/var/run/secrets/kubernetes.io/serviceaccount/ca.crt',
        },
        bearer_token_file: 'bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token',
        relabel_configs: [
          {
            source_labels: [
              '__meta_kubernetes_namespace',
              '__meta_kubernetes_service_name',
              '__meta_kubernetes_endpoint_port_name',
            ],
            action: 'keep',
            regex: 'default;kubernetes;https',
          },
        ],
      },
    ],
  },
}.config
