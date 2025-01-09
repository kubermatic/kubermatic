package falco

import (
	"encoding/json"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type Falco struct {
	Namespace string
	Name      string
}

var DefaultFalco = Falco{
	Namespace: "falco",
	Name:      "falco",
}

func (f *Falco) GetApplication() ([]byte, error) {
	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      f.Name,
			Namespace: f.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   f.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    f.Name,
				Version: "0.37.0",
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

func (f *Falco) FetchData() (name, namespace, key string, names []string) {
	names = []string{}

	key = "app.kubernetes.io/name"
	return f.Name, f.Namespace, key, names
}
