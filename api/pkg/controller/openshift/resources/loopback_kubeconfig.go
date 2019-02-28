package resources

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/cert/triple"
)

const apiserverLoopbackKubeconfigName = "apiserver-loopback-kubeconfig"

type loopbackKubeconfigCreatorData interface {
	GetRootCAWithContext(context.Context) (*triple.KeyPair, error)
	Cluster() *kubermaticv1.Cluster
	GetApiserverExternalNodePort(context.Context) (int32, error)
}

// GetLoopbackKubeconfigCreator is a function to return a secret generator to create a kubeconfig which must only by the openshift-apiserver itself as it uses 127.0.0.1 as address
// It is required because the Apiserver tries to talk to itself before it is ready, hence it
// doesn't appear as valid endpoint on the service
func GetLoopbackKubeconfigCreator(ctx context.Context, data loopbackKubeconfigCreatorData) resources.NamedSecretCreatorGetter {
	return func() (string, resources.SecretCreator) {
		return apiserverLoopbackKubeconfigName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCAWithContext(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster ca: %v", err)
			}
			commonName := "system:openshift-master"
			organizations := []string{"system:masters"}

			port, err := data.GetApiserverExternalNodePort(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get apiserver port: %v", err)
			}

			url := fmt.Sprintf("https://127.0.0.1:%d", port)

			b := se.Data[resources.KubeconfigSecretKey]
			valid, err := resources.IsValidKubeconfig(b, ca.Cert, url, commonName, organizations, data.Cluster().Name)
			if err != nil || !valid {
				if err != nil {
					glog.V(2).Infof("failed to validate existing kubeconfig from %s/%s %v. Regenerating it...", se.Namespace, se.Name, err)
				} else {
					glog.V(2).Infof("invalid/outdated kubeconfig found in %s/%s. Regenerating it...", se.Namespace, se.Name)
				}

				se.Data[resources.KubeconfigSecretKey], err = resources.BuildNewKubeconfigAsByte(ca, url, commonName, organizations, data.Cluster().Name)
				if err != nil {
					return nil, fmt.Errorf("failed to create new kubeconfig: %v", err)
				}
				return se, nil
			}

			return se, nil
		}
	}
}
