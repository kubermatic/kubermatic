local health = import "../../vendor/kubernetes-grafana/src/kubernetes-jsonnet/grafana/configs/dashboard-definitions/kubernetes-cluster-health-dashboard.jsonnet";
local kubermatic = import "../dashboard.jsonnet";

health + kubermatic {
    title: "Cluster Health"
}
