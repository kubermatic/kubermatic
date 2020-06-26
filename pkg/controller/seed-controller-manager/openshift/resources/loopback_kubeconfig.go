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

package resources

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
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
func GetLoopbackKubeconfigCreator(ctx context.Context, data loopbackKubeconfigCreatorData, log *zap.SugaredLogger) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
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
					log.Infow("failed to validate existing kubeconfig. Regenerating it...", "secret-namespace", se.Namespace, "secret-name", se.Name, zap.Error(err))
				} else {
					log.Infow("invalid/outdated kubeconfig found in %s/%s. Regenerating it...", "secret-namespace", se.Namespace, "secret-name", se.Name)
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
