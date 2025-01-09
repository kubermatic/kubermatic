package cert_manager

import (
	"encoding/json"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type Trivy struct {
	Namespace string
	Name      string
}

var DefaultTrivy = Trivy{
	Namespace: "trivy",
	Name:      "trivy",
}

func (t *Trivy) GetApplication() ([]byte, error) {
	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      t.Name,
			Namespace: t.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   t.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    t.Name,
				Version: "0.37.2",
			},
		},
	}
	applications := []apiv1.Application{app}
	data, err := json.Marshal(applications)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (t *Trivy) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"trivy",
	}

	key = "app.kubernetes.io/name"
	return t.Name, t.Namespace, key, names
}
