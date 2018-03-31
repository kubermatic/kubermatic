local deployments = import "../../vendor/kubernetes-grafana/src/kubernetes-jsonnet/grafana/configs/dashboard-definitions/deployments-dashboard.jsonnet";
local kubermatic = import "../dashboard.jsonnet";

deployments + kubermatic
