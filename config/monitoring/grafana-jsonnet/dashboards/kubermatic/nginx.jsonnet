local grafana = import "grafonnet/grafana.libsonnet";
local dashboard = grafana.dashboard;
local graphPanel = grafana.graphPanel;
local prometheus = grafana.prometheus;
local row = grafana.row;
local singlestat = grafana.singlestat;
local kubermaticDashboard = import "../dashboard.jsonnet";

local rateRow = row.new()
    .addPanel(
        graphPanel.new(
            "Rate of total nginx requests per 5min",
            datasource="prometheus",
            min=0,
        )
        .addTarget(prometheus.target(
            "sum(rate(nginx_requests_total[5m])) by (instance)",
            legendFormat="{{ instance }}"
        ))
    );


local activeConnectionsState = singlestat.new(
            "Current Active Connections",
            datasource="prometheus",
            valueName='current',
            span=3,
        )
        .addTarget(prometheus.target(
            "sum(nginx_connnections)",
        )) + {
            sparkline: {
                show: true,
                lineColor: 'rgb(31, 120, 193)',
                fillColor: 'rgba(31, 118, 189, 0.18)',
            },
        };

local activeConnectionsRow = row.new()
    .addPanel(activeConnectionsState)
    .addPanel(
        graphPanel.new(
            "Active Connections",
            datasource="prometheus",
            min=0,
            span=9,
        )
        .addTarget(prometheus.target(
            "sum(nginx_connnections) by (instance)",
            legendFormat="{{ instance }}"
        ))
    );

dashboard.new("Nginx", time_from="now-24h")
    .addRow(rateRow)
    .addRow(activeConnectionsRow)
+ kubermaticDashboard
