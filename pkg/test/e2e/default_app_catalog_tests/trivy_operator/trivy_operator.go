package trivy_operator

import (
	"encoding/json"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type TrivyOperator struct {
	Namespace string
	Name      string
}

var DefaultTrivyOperator = TrivyOperator{
	Namespace: "trivy-operator",
	Name:      "trivy-operator",
}

func (to *TrivyOperator) GetApplication() ([]byte, error) {
	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      to.Name,
			Namespace: to.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   to.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    to.Name,
				Version: "0.18.4",
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

func (to *TrivyOperator) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"trivy-operator",
	}

	key = "app.kubernetes.io/name"
	return to.Name, to.Namespace, key, names
}
