local status = import "../../vendor/kubernetes-grafana/src/kubernetes-jsonnet/grafana/configs/dashboard-definitions/kubernetes-cluster-status-dashboard.jsonnet";
local kubermatic = import "../dashboard.jsonnet";

status + kubermatic {
    title: "Cluster Status"
}
