# kubelb-manager

Helm chart for KubeLB Manager. This is used to deploy the KubeLB CCM to a Kubernetes cluster. The CCM is responsible for propagating the load balancer configurations to the management cluster.

![Version: v1.1.0](https://img.shields.io/badge/Version-v1.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v1.1.0](https://img.shields.io/badge/AppVersion-v1.1.0-informational?style=flat-square)

## Installing the chart

### Pre-requisites

* Create a namespace `kubelb` for the CCM to be deployed in.
* Create imagePullSecrets for the chart to pull the image from the registry.

At this point a minimal values.yaml should look like this:

```yaml
imagePullSecrets:
  - name: <imagePullSecretName>
```

### Install helm chart

Now, we can install the helm chart:

```sh
helm pull oci://quay.io/kubermatic/helm-charts/kubelb-manager-ee --version=v1.1.0 --untardir "kubelb-manager" --untar
## Create and update values.yaml with the required values.
helm install kubelb-manager kubelb-manager --namespace kubelb -f values.yaml --create-namespace
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| autoscaling.enabled | bool | `false` |  |
| autoscaling.maxReplicas | int | `10` |  |
| autoscaling.minReplicas | int | `1` |  |
| autoscaling.targetCPUUtilizationPercentage | int | `80` |  |
| autoscaling.targetMemoryUtilizationPercentage | int | `80` |  |
| cert-manager.enabled | bool | `false` | Enable cert-manager. |
| external-dns.enabled | bool | `false` | Enable External-DNS. |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"quay.io/kubermatic/kubelb-manager-ee"` |  |
| image.tag | string | `"v1.1.0"` |  |
| imagePullSecrets[0].name | string | `"kubermatic-quay.io"` |  |
| kkpintegration.rbac | bool | `false` | Create RBAC for KKP integration. |
| kubelb.debug | bool | `true` |  |
| kubelb.enableGatewayAPI | bool | `false` | enableGatewayAPI specifies whether to enable the Gateway API and Gateway Controllers. By default Gateway API is disabled since without Gateway APIs installed the controller cannot start. |
| kubelb.enableLeaderElection | bool | `true` |  |
| kubelb.enableTenantMigration | bool | `true` |  |
| kubelb.envoyProxy.affinity | object | `{}` |  |
| kubelb.envoyProxy.nodeSelector | object | `{}` |  |
| kubelb.envoyProxy.replicas | int | `3` | The number of replicas for the Envoy Proxy deployment. |
| kubelb.envoyProxy.resources | object | `{}` |  |
| kubelb.envoyProxy.singlePodPerNode | bool | `true` | Deploy single pod per node. |
| kubelb.envoyProxy.tolerations | list | `[]` |  |
| kubelb.envoyProxy.topology | string | `"shared"` | Topology defines the deployment topology for Envoy Proxy. Valid values are: shared and global. |
| kubelb.envoyProxy.useDaemonset | bool | `false` | Use DaemonSet for Envoy Proxy deployment instead of Deployment. |
| kubelb.propagateAllAnnotations | bool | `false` | Propagate all annotations from the LB resource to the LB service. |
| kubelb.propagatedAnnotations | object | `{}` | Allowed annotations that will be propagated from the LB resource to the LB service. |
| kubelb.skipConfigGeneration | bool | `false` | Set to true to skip the generation of the Config CR. Useful when the config CR needs to be managed manually. |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podLabels | object | `{}` |  |
| podSecurityContext.runAsNonRoot | bool | `true` |  |
| podSecurityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| rbac.allowLeaderElectionRole | bool | `true` |  |
| rbac.allowMetricsReaderRole | bool | `true` |  |
| rbac.allowProxyRole | bool | `true` |  |
| rbac.enabled | bool | `true` |  |
| replicaCount | int | `1` |  |
| resources.limits.cpu | string | `"500m"` |  |
| resources.limits.memory | string | `"512Mi"` |  |
| resources.requests.cpu | string | `"100m"` |  |
| resources.requests.memory | string | `"128Mi"` |  |
| securityContext.allowPrivilegeEscalation | bool | `false` |  |
| securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| securityContext.runAsUser | int | `65532` |  |
| service.port | int | `8001` |  |
| service.protocol | string | `"TCP"` |  |
| service.type | string | `"ClusterIP"` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `""` |  |
| serviceMonitor.enabled | bool | `false` |  |
| tolerations | list | `[]` |  |

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Kubermatic | <support@kubermatic.com> | <https://kubermatic.com> |
