local nodes = import "../../vendor/kubernetes-grafana/src/kubernetes-jsonnet/grafana/configs/dashboard-definitions/nodes.jsonnet";
local kubermatic = import "../dashboard.jsonnet";

nodes + kubermatic
