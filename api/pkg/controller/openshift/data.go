package openshift

import (
	"context"
	"fmt"

	kubernetesresources "github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// openshiftData implements the openshiftData interface which is
// passed into all creator funcs and contains all data they need
type openshiftData struct {
	cluster *kubermaticv1.Cluster
	client  client.Client
}

func (od *openshiftData) secretRevision(ctx context.Context, name string) (string, error) {
	secret := &corev1.Secret{}
	if err := od.client.Get(ctx, nn(cluster.Status.NamespaceName, name), secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s: %v", name, err)
	}
	return secret.ResourceVersion, nil
}

func (od *openshiftData) configmapRevision(ctx context.Context, name string) (string, error) {
	configMap := &corev1.ConfigMap{}
	if err := od.client.Get(ctx, nn(cluster.Status.NamespaceName, name), configMap); err != nil {
		return "", fmt.Errorf("failed to get configmap %s: %v", name, err)
	}
	return configMap.ResourceVersion, nil
}

func (od *openshiftData) Cluster(_ context.Context) *kubermaticv1.Cluster {
	return od.cluster
}

func (od *openshiftData) GetPodTemplateLabels(ctx context.Context, appName string, volumes []corev1.Volume, additionalLabels map[string]string) (map[string]string, error) {
	podLabels := kubernetesresources.AppClusterLabel(appName, od.cluster.Name, additionalLabels)
	for _, v := range volumes {
		if v.VolumeSource.Secret != nil {
			revision, err := od.SecretRevision(ctx, v.VolumeSource.Secret.SecretName)
			if err != nil {
				return nil, err
			}
			podLabels[fmt.Sprintf("%s-secret-revision", v.VolumeSource.Secret.SecretName)] = revision
		}
		if v.VolumeSource.ConfigMap != nil {
			revision, err := od.ConfigMapRevision(ctx, v.VolumeSource.ConfigMap.Name)
			if err != nil {
				return nil, err
			}
			podLabels[fmt.Sprintf("%s-configmap-revision", v.VolumeSource.ConfigMap.Name)] = revision
		}
	}

	return podLabels, nil
}

func (od *openshiftData) GetApiserverExternalNodePort(ctx context.Context) (int32, error) {
	service := &corev1.Service{}
	if err := od.client.Get(ctx,
		nn(cluster.Status.NamespaceName, kubernetesresources.ApiserverExternalServiceName),
		service); err != nil {
		return 0, fmt.Errorf("failed to get apiservice for cluster %s: %v", od.Cluster.Name, err)
	}

	if portLen := len(service.Spec.Ports); portLen != 1 {
		return 0, fmt.Errorf("expected service %s to have exactly one port but has %d",
			kubernetesresources.ApiserverExternalServiceName, portLen)
	}
	return service.Spec.Ports[0].NodePort, nil
}

func (od *openshiftData) NodePortRange(_ context.Context) string {
	//TODO: softcode this
	return "30000-32767"
}
