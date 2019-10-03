package migrations

import (
	"testing"

	kubermaticfakeclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateSecretsForCredentials(t *testing.T) {
	tests := []struct {
		name        string
		spec        kubermaticv1.CloudSpec
		secretToken string
	}{
		{
			name: "cluster with only value defined",
			spec: kubermaticv1.CloudSpec{
				Hetzner: &kubermaticv1.HetznerCloudSpec{
					Token: "some-token",
				},
			},
		},
		{
			name:        "cluster with only reference defined",
			secretToken: "some-token",
			spec: kubermaticv1.CloudSpec{
				Hetzner: &kubermaticv1.HetznerCloudSpec{
					CredentialsReference: &providerconfig.GlobalSecretKeySelector{
						ObjectReference: corev1.ObjectReference{
							Name:      "credential-hetzner-test-cluster",
							Namespace: resources.KubermaticNamespace,
						},
						Key: resources.HetznerToken,
					},
				},
			},
		},
		{
			name:        "cluster with both reference and values defined",
			secretToken: "some-other-token",
			spec: kubermaticv1.CloudSpec{
				Hetzner: &kubermaticv1.HetznerCloudSpec{
					CredentialsReference: &providerconfig.GlobalSecretKeySelector{
						ObjectReference: corev1.ObjectReference{
							Name:      "credential-hetzner-test-cluster",
							Namespace: resources.KubermaticNamespace,
						},
						Key: resources.HetznerToken,
					},
					Token: "some-token",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kubeObjs := []runtime.Object{}
			if test.spec.Hetzner.CredentialsReference != nil {
				kubeObjs = append(kubeObjs, newSecret(test.secretToken))
			}
			fakeKubeClient := fake.NewSimpleClientset(kubeObjs...)

			cluster := newCluster(test.spec)
			kubermaticObjs := []runtime.Object{cluster}
			fakeKubermaticClient := kubermaticfakeclientset.NewSimpleClientset(kubermaticObjs...)

			ctx := &cleanupContext{
				kubeClient:       fakeKubeClient,
				kubermaticClient: fakeKubermaticClient,
			}
			if err := createSecretsForCredentials(cluster, ctx); err != nil {
				t.Errorf("%s: error creating secrets for credentials: %v", test.name, err)
			}

			if cluster.Spec.Cloud.Hetzner.CredentialsReference == nil {
				t.Errorf("%s: credentialsReference field is not set", test.name)
			}
			if cluster.Spec.Cloud.Hetzner.Token != "" {
				t.Errorf("%s: token field is not cleared", test.name)
			}

			secret, err := ctx.kubeClient.CoreV1().Secrets(resources.KubermaticNamespace).Get(cluster.GetSecretName(), metav1.GetOptions{})
			if err != nil {
				t.Fatalf("%s: error getting secret %s: %v", test.name, cluster.GetSecretName(), err)
			}
			if secret.Data == nil {
				t.Fatalf("%s: secret %s has empty data", test.name, cluster.GetSecretName())
			}

			token, ok := secret.Data[resources.HetznerToken]
			if !ok {
				t.Fatalf("%s: secret %s does not have %s key", test.name, cluster.GetSecretName(), resources.HetznerToken)
			}
			if string(token) != "some-token" {
				t.Errorf("%s: expected token value in secret: \"some-token\", got: %s", test.name, string(token))
			}
		})
	}
}

func newCluster(cloudSpec kubermaticv1.CloudSpec) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: cloudSpec,
		},
	}
}

func newSecret(token string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "credential-hetzner-test-cluster",
			Namespace: resources.KubermaticNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			resources.HetznerToken: []byte(token),
		},
	}
}
