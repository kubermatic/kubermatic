local kubelet = import "../../vendor/kubernetes-grafana/src/kubernetes-jsonnet/grafana/configs/dashboard-definitions/kubernetes-kubelet-dashboard.jsonnet";
local kubermatic = import "../dashboard.jsonnet";

kubelet + kubermatic {
    refresh: "10s",
}
