local grafana = import "grafonnet/grafana.libsonnet";
local dashboard = grafana.dashboard;
local graphPanel = grafana.graphPanel;
local prometheus = grafana.prometheus;
local row = grafana.row;
local singlestat = grafana.singlestat;
local template = grafana.template;
local kubermaticDashboard = import "../dashboard.jsonnet";

local machinesCountStat = singlestat.new(
        "Current Total Machines",
        datasource="prometheus",
        valueName='current',
        span=3,
    )
    .addTarget(prometheus.target(
        "sum(machine_controller_machines{namespace=~\"$namespace\"})",
    ));

local machinesCountGraph = graphPanel.new(
        "Total Machines",
        datasource="prometheus",
        min=0,
        span=9,
    )
    .addTarget(prometheus.target(
        "sum(machine_controller_machines{namespace=~\"$namespace\"})",
    ));

local machinesRow = row.new()
    .addPanel(machinesCountStat)
    .addPanel(machinesCountGraph);

local nodesCountStat = singlestat.new(
        "Current Total Nodes",
        datasource="prometheus",
        valueName='current',
        span=3,
    )
    .addTarget(prometheus.target(
        "sum(machine_controller_nodes{namespace=~\"$namespace\"})",
    ));

local nodesCountGraph = graphPanel.new(
        "Total Nodes",
        datasource="prometheus",
        min=0,
        span=9,
    )
    .addTarget(prometheus.target(
        "sum(machine_controller_nodes{namespace=~\"$namespace\"})",
    ));

local nodesRow = row.new()
    .addPanel(nodesCountStat)
    .addPanel(nodesCountGraph);

local joinDurationGraph = graphPanel.new(
        "Join Duration Quantiles",
        datasource="prometheus",
        min=0,
        format="seconds",
    )
    .addTarget(prometheus.target(
        "histogram_quantile(0.99, sum(machine_controller_controller_operation_duration_seconds_bucket) by (le,operation))",
        legendFormat="0.99 - {{ operation }}",
    ))
    .addTarget(prometheus.target(
        "histogram_quantile(0.90, sum(machine_controller_controller_operation_duration_seconds_bucket) by (le,operation))",
        legendFormat="0.90 - {{ operation }}",
    ))
    .addTarget(prometheus.target(
        "histogram_quantile(0.50, sum(machine_controller_controller_operation_duration_seconds_bucket) by (le,operation))",
        legendFormat="0.50 - {{ operation }}",
    ));

local durationRow = row.new()
    .addPanel(joinDurationGraph);

local errorsGraph = graphPanel.new(
        "5min Error Rate",
        datasource="prometheus",
        min=0,
        span=6,
    )
    .addTarget(prometheus.target(
        "sum(rate(machine_controller_errors_total{namespace=~\"$namespace\"}[5m])) by (namespace)",
        legendFormat="{{ namespace }}",
    ));

local workersGraph = graphPanel.new(
        "Workers",
        datasource="prometheus",
        min=0,
        span=6,
    )
    .addTarget(prometheus.target(
        "sum(machine_controller_workers{namespace=~\"$namespace\"}) by (namespace)",
        legendFormat="{{ namespace }}",
    ));

local miscRow = row.new()
    .addPanel(errorsGraph)
    .addPanel(workersGraph);

dashboard.new("Machine Controller", time_from="now-24h")
    .addTemplate(
        template.new(
            'namespace',
            'prometheus',
            'label_values(machine_controller_nodes,namespace)',
            refresh='time',
            includeAll=true,
        )
    )
    .addRow(machinesRow)
    .addRow(nodesRow)
    .addRow(miscRow)
+ kubermaticDashboard
