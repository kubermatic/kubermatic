package echoserver

import (
	"encoding/json"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type KubeVirt struct {
	Namespace string
	Name      string
}

var DefaultKubeVirt = KubeVirt{
	Namespace: "kubevirt",
	Name:      "kubevirt",
}

func (kv *KubeVirt) GetApplication() ([]byte, error) {
	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      kv.Name,
			Namespace: kv.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   kv.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    kv.Name,
				Version: "v1.1.0",
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

func (kv *KubeVirt) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"cdi-operator",
		"virt-operator",
	}

	key = "name"
	return kv.Name, kv.Namespace, key, names
}
