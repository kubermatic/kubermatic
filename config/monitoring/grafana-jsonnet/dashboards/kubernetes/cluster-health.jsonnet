local health = import "../../vendor/kubernetes-grafana/src/kubernetes-jsonnet/grafana/configs/dashboard-definitions/kubernetes-cluster-health-dashboard.libsonnet";
local kubermatic = import "../dashboard.jsonnet";

health + kubermatic {
    title: "Cluster Health"
}
