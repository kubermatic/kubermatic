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

func (cm *CertManager) GetApplication() ([]byte, error) {
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
				Version: "v1.14.1",
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

func (cm *CertManager) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"argocd-application-controller",
		"argocd-applicationset-controller",
		"argocd-dex-server",
		"argocd-notifications-controller",
		"argocd-redis",
		"argocd-repo-server",
		"argocd-server",
	}

	key = "app.kubernetes.io/name"

	return cm.Name, cm.Namespace, key, names
}
