package openshift

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/url"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubernetesresources "github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// openshiftData implements the openshiftData interface which is
// passed into all creator funcs and contains all data they need
type openshiftData struct {
	cluster *kubermaticv1.Cluster
	client  client.Client
}

func (od *openshiftData) GetRootCA() (*triple.KeyPair, error) {
	return od.GetRootCAWithContext(context.Background())
}

func (od *openshiftData) GetRootCAWithContext(ctx context.Context) (*triple.KeyPair, error) {
	secret := &corev1.Secret{}
	if err := od.client.Get(ctx, nn(od.cluster.Status.NamespaceName, kubernetesresources.CASecretName), secret); err != nil {
		return nil, fmt.Errorf("failed to get cluster ca: %v", err)
	}
	certs, err := certutil.ParseCertsPEM(secret.Data[kubernetesresources.CACertSecretKey])
	if err != nil {
		return nil, fmt.Errorf("got an invalid cert from the CA secret: %v", err)
	}

	if len(certs) != 1 {
		return nil, fmt.Errorf("did not find exactly one but %v certificates in the CA secret", len(certs))
	}

	key, err := certutil.ParsePrivateKeyPEM(secret.Data[kubernetesresources.CAKeySecretKey])
	if err != nil {
		return nil, fmt.Errorf("got an invalid private key from the CA secret: %v", err)
	}

	rsaKey, isRSAKey := key.(*rsa.PrivateKey)
	if !isRSAKey {
		return nil, errors.New("key is not a RSA key")
	}
	return &triple.KeyPair{Cert: certs[0], Key: rsaKey}, nil
}

// TODO: Implement option to override
func (od *openshiftData) ImageRegistry(registry string) string {
	return registry
}

// TODO: Softcode this, its an arg to the kubermatic controller manager
func (od *openshiftData) NodeAccessNetwork() string {
	return "10.254.0.0/16"
}

func (od *openshiftData) GetClusterRef() metav1.OwnerReference {
	gv := kubermaticv1.SchemeGroupVersion
	return *metav1.NewControllerRef(od.cluster, gv.WithKind("Cluster"))
}

func (od *openshiftData) ClusterIPByServiceName(name string) (string, error) {
	service := &corev1.Service{}
	if err := od.client.Get(context.TODO(), nn(od.cluster.Status.NamespaceName, name), service); err != nil {
		return "", fmt.Errorf("failed to get service %s: %v", name, err)
	}
	return service.Spec.ClusterIP, nil
}

func (od *openshiftData) secretRevision(ctx context.Context, name string) (string, error) {
	secret := &corev1.Secret{}
	if err := od.client.Get(ctx, nn(od.cluster.Status.NamespaceName, name), secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s: %v", name, err)
	}
	return secret.ResourceVersion, nil
}

func (od *openshiftData) configmapRevision(ctx context.Context, name string) (string, error) {
	configMap := &corev1.ConfigMap{}
	if err := od.client.Get(ctx, nn(od.cluster.Status.NamespaceName, name), configMap); err != nil {
		return "", fmt.Errorf("failed to get configmap %s: %v", name, err)
	}
	return configMap.ResourceVersion, nil
}

func (od *openshiftData) Cluster() *kubermaticv1.Cluster {
	return od.cluster
}

func (od *openshiftData) GetPodTemplateLabels(ctx context.Context, appName string, volumes []corev1.Volume, additionalLabels map[string]string) (map[string]string, error) {
	podLabels := kubernetesresources.AppClusterLabel(appName, od.cluster.Name, additionalLabels)
	for _, v := range volumes {
		if v.VolumeSource.Secret != nil {
			revision, err := od.secretRevision(ctx, v.VolumeSource.Secret.SecretName)
			if err != nil {
				return nil, err
			}
			podLabels[fmt.Sprintf("%s-secret-revision", v.VolumeSource.Secret.SecretName)] = revision
		}
		if v.VolumeSource.ConfigMap != nil {
			revision, err := od.configmapRevision(ctx, v.VolumeSource.ConfigMap.Name)
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
		nn(od.cluster.Status.NamespaceName, kubernetesresources.ApiserverExternalServiceName),
		service); err != nil {
		return 0, fmt.Errorf("failed to get apiservice for cluster %s: %v", od.cluster.Name, err)
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

func (od *openshiftData) InClusterApiserverURL() (*url.URL, error) {
	// We have to fullfull the templateData interface here which doesn't have a context as arg
	// Needed for pkg/resources/apiserver.IsRunningInitContainer
	ctx := context.TODO()
	port, err := od.GetApiserverExternalNodePort(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get apiserver nodeport: %v", err)
	}
	dnsName := kubernetesresources.GetAbsoluteServiceDNSName(kubernetesresources.ApiserverExternalServiceName, od.cluster.Status.NamespaceName)
	return url.Parse(fmt.Sprintf("https://%s:%d", dnsName, port))
}
