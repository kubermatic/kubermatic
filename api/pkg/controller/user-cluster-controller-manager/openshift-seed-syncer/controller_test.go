package openshiftseedsyncer

import (
	"crypto/x509"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// TestSeedAdminKubeconfigSecretCreatorGetterAddsTokenKey verifies the
// seedAdminKubeconfigSecretCreatorGetter puts the token in both the kubeconfig
// and under a key named `token`. We need that for the openshift console which
// is hardcoded to only function with a token read from
// `/var/run/secrets/kubernetes.io/serviceaccount/token`, it doesn't work with any
// other kind of authentication like certs.
func TestSeedAdminKubeconfigSecretCreatorGetterAddsTokenKey(t *testing.T) {
	cert := &x509.Certificate{}
	apiServerAddress := "https://my-api:123"
	clusterName := "my-cluster"
	token := "my-token"

	_, creator := seedAdminKubeconfigSecretCreatorGetter(cert, apiServerAddress, clusterName, token)()
	secret, err := creator(&corev1.Secret{})
	if err != nil {
		t.Fatalf("error calling creator: %v", err)
	}

	if _, exists := secret.Data[resources.KubeconfigSecretKey]; !exists {
		t.Fatalf("Secret has no %q key", resources.KubeconfigSecretKey)
	}
	cfg, err := clientcmd.Load(secret.Data[resources.KubeconfigSecretKey])
	if err != nil {
		t.Fatalf("failed to load kubeconfig from secret: %v", err)
	}
	if val := cfg.AuthInfos[resources.KubeconfigDefaultContextKey].Token; val != token {
		t.Errorf("expected kubeconfig token to be %q, was %q", token, val)
	}
	val, exists := secret.Data["token"]
	if !exists {
		t.Fatal("Secret has no `token` key")
	}
	if string(val) != token {
		t.Errorf("Value of token key %q does not match expected %q", string(val), token)
	}

}
