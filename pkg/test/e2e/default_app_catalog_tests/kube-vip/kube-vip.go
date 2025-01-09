package kube_vip

import (
	"encoding/json"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type KubeVip struct {
	Namespace string
	Name      string
}

var DefaultKubeVip = KubeVip{
	Namespace: "kube-vip",
	Name:      "kube-vip",
}

func (kv *KubeVip) GetApplication() ([]byte, error) {
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
				Version: "v0.4.1",
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

func (kv *KubeVip) FetchData() (name, namespace, key string, names []string) {
	names = []string{}

	key = "app.kubernetes.io/name"
	return kv.Name, kv.Namespace, key, names
}
