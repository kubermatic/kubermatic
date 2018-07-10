package etcd

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

// TLSCertificate returns a secret with the etcd tls certificate
func TLSCertificate(data *resources.TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
	var se *corev1.Secret
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Secret{}
	}

	se.Name = resources.EtcdTLSCertificateSecretName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	ca, err := data.GetClusterCA()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster ca: %v", err)
	}

	altNames := sets.NewString("127.0.0.1", "localhost", fmt.Sprintf("%s-%s.svc.cluster.local.", resources.EtcdClientServiceName, data.Cluster.Name))
	for i := 0; i < 3; i++ {
		// Member name
		podName := fmt.Sprintf("etcd-%d", i)
		altNames.Insert(podName)

		// Pod DNS name
		absolutePodDNSName := fmt.Sprintf("etcd-%d.%s.%s.svc.cluster.local", i, resources.EtcdServiceName, data.Cluster.Status.NamespaceName)
		altNames.Insert(absolutePodDNSName)
	}

	clientIP, err := data.ClusterIPByServiceName(resources.EtcdClientServiceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get ClientIP of etcd client service '%s': %v", resources.EtcdClientServiceName, err)
	}

	if b, exists := se.Data[resources.EtcdTLSCertSecretKey]; exists {
		certs, err := certutil.ParseCertsPEM(b)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret %s: %v", resources.EtcdTLSCertSecretKey, resources.EtcdTLSCertificateSecretName, err)
		}

		if resources.IsServerCertificateValidForAllOf(certs[0], "etcd", resources.EtcdClientServiceName, data.Cluster.Status.NamespaceName, "cluster.local", []string{clientIP}, altNames.List()) {
			return se, nil
		}
	}

	newKP, err := triple.NewServerKeyPair(ca, "etcd", resources.EtcdClientServiceName, data.Cluster.Status.NamespaceName, "cluster.local", []string{clientIP}, altNames.List())
	if err != nil {
		return nil, fmt.Errorf("failed to create apiserver key pair: %v", err)
	}

	se.Data = map[string][]byte{
		resources.EtcdTLSKeySecretKey:  certutil.EncodePrivateKeyPEM(newKP.Key),
		resources.EtcdTLSCertSecretKey: certutil.EncodeCertPEM(newKP.Cert),
	}

	return se, nil
}
