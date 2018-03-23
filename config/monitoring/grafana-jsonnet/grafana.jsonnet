local k = import "ksonnet.beta.3/k.libsonnet";
local configMap = k.core.v1.configMap;

local dashboardSources = import "dashboards/kubernetes/kubernetes.jsonnet";

local dashboards = {
    "capacity-planning.json": import "dashboards/kubernetes/capacity-planning.jsonnet",
    "cluster-health.json": import "dashboards/kubernetes/cluster-health.jsonnet",
    "cluster-status.json": import "dashboards/kubernetes/cluster-status.jsonnet",
    "deployments.json": import "dashboards/kubernetes/deployments.jsonnet",
    "kubernetes-kubelet.json": import "dashboards/kubernetes/kubelet.jsonnet",
    "kubernetes-nodes.json": import "dashboards/kubernetes/nodes.jsonnet",
    "pods.json": import "dashboards/kubernetes/pods.jsonnet",
};

k.core.v1.list.new([
    configMap.new("grafana-dashboards", { "dashboards.yaml": std.manifestJsonEx(dashboardSources, "    ") }),
    configMap.new("grafana-datasources", { "prometheus.yaml": std.manifestJsonEx(import "prometheus.jsonnet", "    ") }),
    configMap.new("grafana-dashboard-definitions", { [name]: std.manifestJsonEx(dashboards[name], "    ") for name in std.objectFields(dashboards) }),
])
