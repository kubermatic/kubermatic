package cert_manager

import (
	"encoding/json"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type CertManager struct {
	Namespace string
	Name      string
}

var DefaultCertManager = CertManager{
	Namespace: "cert-manager",
	Name:      "cert-manager",
}

func (cm *CertManager) GetApplication(version string) ([]byte, error) {
	valuesBlock := `installCRDs: true`

	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      cm.Name,
			Namespace: cm.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   cm.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    cm.Name,
				Version: version,
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

func (cm *CertManager) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"cert-manager",
		"cert-manager-cainjector",
		"cert-manager-startupapicheck",
		"cert-manager-webhook",
	}

	key = "app.kubernetes.io/name"
	return cm.Name, cm.Namespace, key, names
}
