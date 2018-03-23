{
    datasources: [
        {
            name: "prometheus",
            type: "prometheus",
            access: "proxy",
            org_id: 1,
            url: "http://prometheus.monitoring.svc.cluster.local:9090",
            version: 1,
            editable: false,
        },
    ],
}
