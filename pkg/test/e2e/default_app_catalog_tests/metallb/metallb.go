package metallb

import (
	"encoding/json"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type MetalLB struct {
	Namespace string
	Name      string
}

var DefaultMetalLB = MetalLB{
	Namespace: "metallb",
	Name:      "metallb",
}

func (mlb *MetalLB) GetApplication(version string) ([]byte, error) {
	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      mlb.Name,
			Namespace: mlb.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   mlb.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    mlb.Name,
				Version: version,
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

func (mlb *MetalLB) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"metallb",
	}

	key = "app.kubernetes.io/name"
	return mlb.Name, mlb.Namespace, key, names
}
