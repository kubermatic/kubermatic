local k = import "ksonnet.beta.3/k.libsonnet";
local configMap = k.core.v1.configMap;

local dashboardSources = import "dashboards/sources.jsonnet";

local kubermaticDashboards = {
    "machine-controller.json": import "dashboards/kubermatic/machine-controller.jsonnet",
    "nginx.json": import "dashboards/kubermatic/nginx.jsonnet",
};

local kubernetesDashboards = {
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
    configMap.new("grafana-dashboards-kubermatic", { [name]: std.manifestJsonEx(kubermaticDashboards[name], "    ") for name in std.objectFields(kubermaticDashboards) }),
    configMap.new("grafana-dashboards-kubernetes", { [name]: std.manifestJsonEx(kubernetesDashboards[name], "    ") for name in std.objectFields(kubernetesDashboards) }),
])
