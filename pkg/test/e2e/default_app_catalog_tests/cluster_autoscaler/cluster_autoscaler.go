package cluster_autoscaler

import (
	"encoding/json"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type ClusterAutoScaler struct {
	Namespace string
	Name      string
}

var DefaultCertManager = ClusterAutoScaler{
	Namespace: "cluster-autoscaler",
	Name:      "cluster-autoscaler",
}

func (ca *ClusterAutoScaler) GetApplication() ([]byte, error) {
	valuesBlock := `cloudProvider: clusterapi
clusterAPIMode: incluster-incluster
autoDiscovery:
  namespace: kube-system
image:
  tag: '{{ .Cluster.AutoscalerVersion }}'
extraEnv:
  CAPI_GROUP: cluster.k8s.io
rbac:
  create: true
  pspEnabled: false
  clusterScoped: true
  serviceAccount:
    annotations: {}
    create: true
    name: "cluster-autoscaler-clusterapi-cluster-autoscaler"
    automountServiceAccountToken: true
extraObjects:
- apiVersion: rbac.authorization.k8s.io/v1
  kind: Role
  metadata:
    name: cluster-autoscaler-management
    namespace: kube-system
  rules:
  - apiGroups:
    - cluster.k8s.io
    resources:
    - machinedeployments
    - machinedeployments/scale
    - machines
    - machinesets
    verbs:
    - get
    - list
    - update
    - watch
- apiVersion: rbac.authorization.k8s.io/v1
  kind: RoleBinding
  metadata:
    name: cluster-autoscaler-clusterapi-cluser-autoscaler
    namespace: kube-system
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: Role
    name: cluster-autoscaler-management
    namespace: kube-system
  subjects:
  - kind: ServiceAccount
    name: cluster-autoscaler-clusterapi-cluster-autoscaler
    namespace: kube-system`

	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      ca.Name,
			Namespace: ca.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   ca.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    ca.Name,
				Version: "1.31.0",
			},
			ValuesBlock: valuesBlock,
		},
	}
	applications := []apiv1.Application{app}
	data, err := json.Marshal(applications)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (ca *ClusterAutoScaler) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"cert-manager",
		"cert-manager-cainjector",
		"cert-manager-startupapicheck",
		"cert-manager-webhook",
	}

	key = "app.kubernetes.io/name"
	return ca.Name, ca.Namespace, key, names
}
