package echoserver

import (
	"encoding/json"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type Flux struct {
	Namespace string
	Name      string
}

var DefaultFlux = Flux{
	Namespace: "flux2",
	Name:      "flux2",
}

func (flux *Flux) GetApplication() ([]byte, error) {
	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      flux.Name,
			Namespace: flux.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   flux.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    flux.Name,
				Version: "2.2.2",
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

func (flux *Flux) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"flux2",
	}

	key = "app.kubernetes.io/instance"
	return flux.Name, flux.Namespace, key, names
}
