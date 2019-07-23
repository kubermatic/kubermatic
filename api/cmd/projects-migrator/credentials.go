package main

import (
	"fmt"

	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	provider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func createSecretsForCredentials(cluster *kubermaticv1.Cluster, client kubernetes.Interface) error {
	if cluster.Spec.Cloud.Packet != nil {
		return createPacketSecret(cluster, client)
	}
	return nil
}

func createPacketSecret(cluster *kubermaticv1.Cluster, client kubernetes.Interface) error {
	packetSpec := cluster.Spec.Cloud.Packet
	referenceDefined := packetSpec.APIKeyReference != nil && packetSpec.ProjectIDReference != nil
	valuesDefined := packetSpec.APIKey != "" && packetSpec.ProjectID != ""
	if !referenceDefined && valuesDefined {
		packetSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-packet-%s", provider.CredentialPrefix, cluster.Name),
				Namespace: resources.KubermaticNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"apiKey":    []byte(packetSpec.APIKey),
				"projectID": []byte(packetSpec.ProjectID),
			},
		}

		if _, err := client.CoreV1().Secrets(resources.KubermaticNamespace).Create(packetSecret); err != nil {
			return err
		}
		glog.Infof("the secret = %s was created for cluster = %s resource", packetSecret.Name, cluster.Name)
	}
	return nil
}
