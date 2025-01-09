package nginx_ingress_controller

import (
	"encoding/json"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type NginxIngressController struct {
	Namespace string
	Name      string
}

var DefaultNginxIngressController = NginxIngressController{
	Namespace: "nginx",
	Name:      "nginx",
}

func (nic *NginxIngressController) GetApplication() ([]byte, error) {
	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      nic.Name,
			Namespace: nic.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   nic.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    nic.Name,
				Version: "1.9.6",
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

func (nic *NginxIngressController) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"ingress-nginx",
	}

	key = "app.kubernetes.io/name"
	return nic.Name, nic.Namespace, key, names
}
