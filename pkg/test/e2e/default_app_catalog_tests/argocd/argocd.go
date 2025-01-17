package argocd

import (
	"encoding/json"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type ArgoCD struct {
	Namespace string
	Name      string
}

var DefaultArgoCD = ArgoCD{
	Namespace: "argocd",
	Name:      "argocd",
}

func (a *ArgoCD) GetApplication() ([]byte, error) {
	valuesBlock := `
server:
  service:
    type: "LoadBalancer"`

	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      a.Name,
			Namespace: a.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   a.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    a.Name,
				Version: "v2.10.0",
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

func (a *ArgoCD) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"cluster-autoscaler",
	}

	key = "app.kubernetes.io/name"

	return a.Name, a.Namespace, key, names
}
