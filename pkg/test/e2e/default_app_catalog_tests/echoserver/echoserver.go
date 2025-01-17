package echoserver

import (
	"encoding/json"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

type EchoServer struct {
	Namespace string
	Name      string
}

var DefaultEchoServer = EchoServer{
	Namespace: "echoserver",
	Name:      "echoserver",
}

func (es *EchoServer) GetApplication(version string) ([]byte, error) {
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

func (es *EchoServer) FetchData() (name, namespace, key string, names []string) {
	names = []string{
		"echo-server",
	}

	key = "app.kubernetes.io/name"
	return es.Name, es.Namespace, key, names
}
