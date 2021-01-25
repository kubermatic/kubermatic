/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package openshift

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/openshift/resources"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesresources "k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	certutil "k8s.io/client-go/util/cert"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// openshiftData implements the openshiftData interface which is
// passed into all creator funcs and contains all data they need
type openshiftData struct {
	cluster                               *kubermaticv1.Cluster
	client                                ctrlruntimeclient.Client
	dc                                    *kubermaticv1.Datacenter
	overwriteRegistry                     string
	nodeAccessNetwork                     string
	oidc                                  OIDCConfig
	etcdDiskSize                          resource.Quantity
	etcdLauncherImage                     string
	kubermaticImage                       string
	dnatControllerImage                   string
	supportsFailureDomainZoneAntiAffinity bool
	externalURL                           string
	seed                                  *kubermaticv1.Seed
	versions                              kubermatic.Versions
}

func (od *openshiftData) DC() *kubermaticv1.Datacenter {
	return od.dc
}

func (od *openshiftData) GetOpenVPNCA() (*kubernetesresources.ECDSAKeyPair, error) {
	return od.GetOpenVPNCAWithContext(context.TODO())
}

func (od *openshiftData) GetOpenVPNCAWithContext(ctx context.Context) (*kubernetesresources.ECDSAKeyPair, error) {
	caCertSecret := &corev1.Secret{}
	if err := od.client.Get(ctx, nn(od.cluster.Status.NamespaceName, kubernetesresources.OpenVPNCASecretName), caCertSecret); err != nil {
		return nil, fmt.Errorf("failed to get OpenVPN CA: %v", err)
	}
	certs, err := certutil.ParseCertsPEM(caCertSecret.Data[kubernetesresources.CACertSecretKey])
	if err != nil {
		return nil, fmt.Errorf("got an invalid cert from the CA secret %s: %v", kubernetesresources.CASecretName, err)
	}

	if len(certs) != 1 {
		return nil, fmt.Errorf("did not find exactly one but %v certificates in the CA secret", len(certs))
	}

	key, err := triple.ParsePrivateKeyPEM(caCertSecret.Data[kubernetesresources.CAKeySecretKey])
	if err != nil {
		return nil, fmt.Errorf("got an invalid private key from the CA secret %s: %v", kubernetesresources.CASecretName, err)
	}

	ecdsaKey, isECDSAKey := key.(*ecdsa.PrivateKey)
	if !isECDSAKey {
		return nil, errors.New("key is not a ECDSA key")
	}
	return &kubernetesresources.ECDSAKeyPair{Cert: certs[0], Key: ecdsaKey}, nil
}

func (od *openshiftData) GetRootCA() (*triple.KeyPair, error) {
	return od.GetRootCAWithContext(context.Background())
}

func (od *openshiftData) GetRootCAWithContext(ctx context.Context) (*triple.KeyPair, error) {
	secret := &corev1.Secret{}
	if err := od.client.Get(ctx, nn(od.cluster.Status.NamespaceName, kubernetesresources.CASecretName), secret); err != nil {
		return nil, fmt.Errorf("failed to get cluster ca: %v", err)
	}
	return triple.ParseRSAKeyPair(secret.Data[kubernetesresources.CACertSecretKey],
		secret.Data[kubernetesresources.CAKeySecretKey])
}

func (od *openshiftData) GetFrontProxyCA() (*triple.KeyPair, error) {
	return od.GetFrontProxyCAWithContext(context.TODO())
}

func (od *openshiftData) GetFrontProxyCAWithContext(ctx context.Context) (*triple.KeyPair, error) {
	secret := &corev1.Secret{}
	if err := od.client.Get(ctx, nn(od.cluster.Status.NamespaceName, kubernetesresources.FrontProxyCASecretName), secret); err != nil {
		return nil, fmt.Errorf("failed to get FrontProxy CA: %v", err)
	}
	return triple.ParseRSAKeyPair(secret.Data[kubernetesresources.CACertSecretKey],
		secret.Data[kubernetesresources.CAKeySecretKey])
}

func (od *openshiftData) ImageRegistry(registry string) string {
	if od.overwriteRegistry != "" {
		return od.overwriteRegistry
	}
	return registry
}

func (od *openshiftData) NodeAccessNetwork() string {
	if od.nodeAccessNetwork != "" {
		return od.nodeAccessNetwork
	}
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
		if kerrors.IsNotFound(err) {
			// "-1" is not allowed, label values must start and end with an alphanumeric character
			return "1-1", nil
		}
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

func (od *openshiftData) GetPodTemplateLabels(appName string, volumes []corev1.Volume, additionalLabels map[string]string) (map[string]string, error) {
	return od.GetPodTemplateLabelsWithContext(context.TODO(), appName, volumes, additionalLabels)
}

func (od *openshiftData) GetPodTemplateLabelsWithContext(ctx context.Context, appName string, volumes []corev1.Volume, additionalLabels map[string]string) (map[string]string, error) {
	podLabels := kubernetesresources.AppClusterLabels(appName, od.cluster.Name, additionalLabels)
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
	err := od.client.Get(ctx, nn(od.cluster.Status.NamespaceName, kubernetesresources.ApiserverServiceName), service)
	if err != nil {
		return 0, fmt.Errorf("failed to get apiservice for cluster %s: %v", od.cluster.Name, err)
	}

	if portLen := len(service.Spec.Ports); portLen != 1 {
		return 0, fmt.Errorf("expected service %s to have exactly one port but has %d",
			kubernetesresources.ApiserverServiceName, portLen)
	}
	return service.Spec.Ports[0].NodePort, nil
}

func (od *openshiftData) NodePortRange(_ context.Context) string {
	//TODO: softcode this
	return "30000-32767"
}

func (od *openshiftData) GetOpenVPNServerPort() (int32, error) {
	ctx := context.Background()
	service := &corev1.Service{}
	err := od.client.Get(ctx, nn(od.cluster.Status.NamespaceName, kubernetesresources.OpenVPNServerServiceName), service)
	if err != nil {
		return 0, fmt.Errorf("failed to get apiservice for cluster %s: %v", od.cluster.Name, err)
	}

	if portLen := len(service.Spec.Ports); portLen != 1 {
		return 0, fmt.Errorf("expected service %s to have exactly one port but has %d", kubernetesresources.OpenVPNServerServiceName, portLen)
	}
	return service.Spec.Ports[0].NodePort, nil
}

// GetDexCA returns the chain of public certificates of the Dex
func (od *openshiftData) GetDexCA() ([]*x509.Certificate, error) {
	return kubernetesresources.GetDexCAFromFile(od.oidc.CAFile)
}

func (od *openshiftData) EtcdDiskSize() resource.Quantity {
	return od.etcdDiskSize
}

func (od *openshiftData) EtcdLauncherImage() string {
	imageSplit := strings.Split(od.etcdLauncherImage, "/")
	var registry, imageWithoutRegistry string
	if len(imageSplit) != 3 {
		registry = kubernetesresources.RegistryDocker
		imageWithoutRegistry = strings.Join(imageSplit, "/")
	} else {
		registry = imageSplit[0]
		imageWithoutRegistry = strings.Join(imageSplit[1:], "/")
	}
	return od.ImageRegistry(registry) + "/" + imageWithoutRegistry
}

func (od *openshiftData) EtcdLauncherTag() string {
	return od.versions.Kubermatic
}

func (od *openshiftData) NodePortProxyTag() string {
	return od.versions.Kubermatic
}

// Openshift has its own DNS cache, so this is always false
func (od *openshiftData) NodeLocalDNSCacheEnabled() bool {
	return false
}

func (od *openshiftData) KubermaticAPIImage() string {
	apiImageSplit := strings.Split(od.kubermaticImage, "/")
	var registry, imageWithoutRegistry string
	if len(apiImageSplit) != 3 {
		registry = kubernetesresources.RegistryDocker
		imageWithoutRegistry = strings.Join(apiImageSplit, "/")
	} else {
		registry = apiImageSplit[0]
		imageWithoutRegistry = strings.Join(apiImageSplit[1:], "/")
	}
	return od.ImageRegistry(registry) + "/" + imageWithoutRegistry
}

func (od *openshiftData) KubermaticDockerTag() string {
	return od.versions.Kubermatic
}

func (od *openshiftData) GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	return provider.SecretKeySelectorValueFuncFactory(context.Background(), od.client)(configVar, key)
}

func (od *openshiftData) DNATControllerImage() string {
	dnatControllerImageSplit := strings.Split(od.dnatControllerImage, "/")
	var registry, imageWithoutRegistry string
	if len(dnatControllerImageSplit) != 3 {
		registry = kubernetesresources.RegistryDocker
		imageWithoutRegistry = strings.Join(dnatControllerImageSplit, "/")
	} else {
		registry = dnatControllerImageSplit[0]
		imageWithoutRegistry = strings.Join(dnatControllerImageSplit[1:], "/")
	}
	return od.ImageRegistry(registry) + "/" + imageWithoutRegistry
}

func (od *openshiftData) DNATControllerTag() string {
	return od.versions.Kubermatic
}

func (od *openshiftData) SupportsFailureDomainZoneAntiAffinity() bool {
	return od.supportsFailureDomainZoneAntiAffinity
}

func (od *openshiftData) GetOauthExternalNodePort() (int32, error) {
	svc := &corev1.Service{}
	name := types.NamespacedName{Namespace: od.cluster.Status.NamespaceName, Name: resources.OauthName}
	if err := od.client.Get(context.Background(), name, svc); err != nil {
		return 0, fmt.Errorf("failed to get service: %v", err)
	}
	if n := len(svc.Spec.Ports); n != 1 {
		return 0, fmt.Errorf("expected service to have exactly one port, had %d", n)
	}
	return svc.Spec.Ports[0].NodePort, nil
}

func (od *openshiftData) ExternalURL() string {
	return od.externalURL
}

func (od *openshiftData) GetKubernetesCloudProviderName() string {
	return kubernetesresources.GetKubernetesCloudProviderName(od.Cluster())
}

func (od *openshiftData) CloudCredentialSecretTemplate() ([]byte, error) {
	// TODO: Support more providers than just AWS :)
	if od.Cluster().Spec.Cloud.AWS == nil {
		return nil, nil
	}
	credentials, err := kubernetesresources.GetCredentials(od)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %v", err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			// https://github.com/openshift/cloud-credential-operator/blob/ec6f38d73a7921e79d0ca7555da3a864e808e681/pkg/aws/actuator/actuator.go#L51
			Name: "aws-creds",
		},
		// https://github.com/openshift/cloud-credential-operator/blob/ec6f38d73a7921e79d0ca7555da3a864e808e681/pkg/aws/actuator/actuator.go#L671-L682
		Data: map[string][]byte{
			"aws_access_key_id":     []byte(credentials.AWS.AccessKeyID),
			"aws_secret_access_key": []byte(credentials.AWS.SecretAccessKey),
		},
	}

	serializedSecret, err := json.Marshal(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret: %v", err)
	}
	return serializedSecret, nil
}

func (od *openshiftData) Seed() *kubermaticv1.Seed {
	return od.seed
}
