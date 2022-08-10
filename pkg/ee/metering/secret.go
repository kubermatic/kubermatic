//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package metering

import (
	"context"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// meteringNamespaceCreator creates the image pull secret in the metering namespace.
func meteringPullSecretCreator(ctx context.Context, client ctrlruntimeclient.Client) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.ImagePullSecretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			var kubermaticPullSecret corev1.Secret
			if err := client.Get(ctx, types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: resources.ImagePullSecretName}, &kubermaticPullSecret); err != nil {
				if err != nil {
					return nil, err
				}
			}
			secret.Type = corev1.SecretTypeDockerConfigJson
			secret.Data = kubermaticPullSecret.Data

			return secret, nil
		}
	}
}

// meteringNamespaceCreator creates the s3 secret in the metering namespace.
func meterings3SecretCreator(ctx context.Context, client ctrlruntimeclient.Client) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return SecretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			var kubermaticPullSecret corev1.Secret
			if err := client.Get(ctx, types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: SecretName}, &kubermaticPullSecret); err != nil {
				if err != nil {
					return nil, err
				}
			}
			secret.Type = corev1.SecretTypeOpaque
			secret.Data = kubermaticPullSecret.Data

			return secret, nil
		}
	}
}
