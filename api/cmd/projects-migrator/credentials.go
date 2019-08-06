package main

import (
	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func createSecretsForCredentials(cluster *kubermaticv1.Cluster, client kubernetes.Interface) error {
	if cluster.Spec.Cloud.AWS != nil {
		return createAWSSecret(cluster, client)
	}
	if cluster.Spec.Cloud.Packet != nil {
		return createPacketSecret(cluster, client)
	}

	return nil
}

func createAWSSecret(cluster *kubermaticv1.Cluster, client kubernetes.Interface) error {
	spec := cluster.Spec.Cloud.AWS
	referenceDefined := spec.CredentialsReference != nil
	valuesDefined := spec.AccessKeyID != "" && spec.SecretAccessKey != ""
	var secret *corev1.Secret
	if !referenceDefined && valuesDefined {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.AWSAccessKeyID:     []byte(spec.AccessKeyID),
				resources.AWSSecretAccessKey: []byte(spec.SecretAccessKey),
			},
		}

		if _, err := client.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret); err != nil {
			return err
		}
		glog.Infof("the secret = %s was created for cluster = %s resource", secret.Name, cluster.Name)
	}

	if !referenceDefined {
		cluster.Spec.Cloud.AWS.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.AWS.AccessKeyID = ""
		cluster.Spec.Cloud.AWS.SecretAccessKey = ""
	}

	return nil
}

func createPacketSecret(cluster *kubermaticv1.Cluster, client kubernetes.Interface) error {
	packetSpec := cluster.Spec.Cloud.Packet
	referenceDefined := packetSpec.CredentialsReference != nil
	valuesDefined := packetSpec.APIKey != "" && packetSpec.ProjectID != ""
	var secret *corev1.Secret
	if !referenceDefined && valuesDefined {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.GetSecretName(),
				Namespace: resources.KubermaticNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				resources.PacketAPIKey:    []byte(packetSpec.APIKey),
				resources.PacketProjectID: []byte(packetSpec.ProjectID),
			},
		}

		if _, err := client.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret); err != nil {
			return err
		}
		glog.Infof("the secret = %s was created for cluster = %s resource", secret.Name, cluster.Name)
	}

	if !referenceDefined {
		cluster.Spec.Cloud.Packet.CredentialsReference = &providerconfig.GlobalSecretKeySelector{
			ObjectReference: corev1.ObjectReference{
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
		}
	}

	if valuesDefined {
		cluster.Spec.Cloud.Packet.APIKey = ""
		cluster.Spec.Cloud.Packet.ProjectID = ""
	}

	return nil
}
