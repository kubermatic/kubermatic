local pods = import "../../vendor/kubernetes-grafana/src/kubernetes-jsonnet/grafana/configs/dashboard-definitions/pods-dashboard.jsonnet";
local kubermatic = import "../dashboard.jsonnet";

pods + kubermatic

