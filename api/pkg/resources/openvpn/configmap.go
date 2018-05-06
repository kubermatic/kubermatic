package openvpn

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Name = "openvpn-client-configs"
)

func ConfigMap(data *resources.TemplateData) (*corev1.ConfigMap, error) {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            Name,
			OwnerReferences: []metav1.OwnerReference{data.GetClusterRef()},
		},
		Data: map[string]string{
			"user-cluster-client": config,
		},
	}, nil
}

const (
	config = `
iroute 172.25.0.0 255.255.0.0
iroute 10.10.10.0 255.255.255.0
`
)
