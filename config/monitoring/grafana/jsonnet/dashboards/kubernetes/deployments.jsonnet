local deployments = import "../../vendor/kubernetes-grafana/src/kubernetes-jsonnet/grafana/configs/dashboard-definitions/deployments-dashboard.libsonnet";
local kubermatic = import "../dashboard.jsonnet";

deployments + kubermatic
