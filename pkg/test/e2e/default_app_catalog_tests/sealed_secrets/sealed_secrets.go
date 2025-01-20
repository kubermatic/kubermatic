package sealed_secrets

import (
	"encoding/json"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type SealedSecrets struct {
	Namespace string
	Name      string
}

var DefaultSealedSecrets = SealedSecrets{
	Namespace: "sealed-secrets",
	Name:      "sealed-secrets",
}

func (ss *SealedSecrets) GetApplication(version string) ([]byte, error) {
	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      ss.Name,
			Namespace: ss.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   ss.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    ss.Name,
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

func (ss *SealedSecrets) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"sealed-secrets",
	}

	key = "app.kubernetes.io/name"
	return ss.Name, ss.Namespace, key, names
}
