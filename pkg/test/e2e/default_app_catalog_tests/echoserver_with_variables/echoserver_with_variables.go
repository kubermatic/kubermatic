package echoserver_with_variables

import (
	"encoding/json"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type EchoServerWithVariables struct {
	Namespace string
	Name      string
}

var DefaultEchoServerWithVariables = EchoServerWithVariables{
	Namespace: "echoserver-with-variables",
	Name:      "echoserver-with-variables",
}

func (es *EchoServerWithVariables) GetApplication() ([]byte, error) {
	valuesBlock := `foo: '{{ .Cluster.Version }}'
ingress:
  enabled: true
  hosts:
    - paths:
        - /
  ingressClassName: nginx
replicaCount: 3`

	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name:      es.Name,
			Namespace: es.Namespace,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:   es.Namespace,
				Create: true,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    es.Name,
				Version: "0.7.0",
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

func (es *EchoServerWithVariables) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"echo-server",
	}

	key = "app.kubernetes.io/name"
	return es.Name, es.Namespace, key, names
}
