local capacity = import "../../vendor/kubernetes-grafana/src/kubernetes-jsonnet/grafana/configs/dashboard-definitions/kubernetes-capacity-planning-dashboard.jsonnet";
local kubermatic = import "../dashboard.jsonnet";

capacity + kubermatic {
    title: "Capacity Planning"
}
