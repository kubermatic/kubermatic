//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2023 Kubermatic GmbH

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

package userclusterresources

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/ee/cluster-backup/storage-location/backupstore/aws"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SecretReconciler returns a function to create the Secret containing the backup destination credentials.
func SecretReconciler(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, cbsl *kubermaticv1.ClusterBackupStorageLocation) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return CloudCredentialsSecretName, func(cm *corev1.Secret) (*corev1.Secret, error) {
			key := types.NamespacedName{Name: cbsl.Spec.Credential.Name, Namespace: resources.KubermaticNamespace}

			secret := &corev1.Secret{}
			if err := client.Get(ctx, key, secret); err != nil {
				return nil, fmt.Errorf("failed to get backup destination credentials secret: %w", err)
			}

			awsAccessKeyId := secret.Data[aws.AccessKeyIDKeyName]
			awsSecretAccessKey := secret.Data[aws.SecretAccessKeyName]
			if awsAccessKeyId == nil || awsSecretAccessKey == nil {
				return nil, fmt.Errorf("backup destination credentials secret is not set correctly: [%s] and [%s] can't be empty", aws.AccessKeyIDKeyName, aws.SecretAccessKeyName)
			}

			cloudCredsFile, err := getVeleroCloudCredentials(awsAccessKeyId, awsSecretAccessKey)
			if err != nil {
				return nil, fmt.Errorf("failed to generate Velero cloud-credentials file: %w", err)
			}
			cm.Data = map[string][]byte{
				defaultCloudCredentialsSecretKeyName: cloudCredsFile,
			}
			return cm, nil
		}
	}
}

var credentialsTemplate string = `[default]
aws_access_key_id = {{ .awsAccessKeyId }}
aws_secret_access_key = {{ .awsSecretAccessKey }}
`

func getVeleroCloudCredentials(awsAccessKeyId, awsSecretAccessKey []byte) ([]byte, error) {
	t, err := template.New("cloud-credentials").Parse(credentialsTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials file template: %w", err)
	}
	var buff bytes.Buffer
	if err := t.Execute(&buff, map[string]interface{}{
		"awsAccessKeyId":     string(awsAccessKeyId),
		"awsSecretAccessKey": string(awsSecretAccessKey),
	}); err != nil {
		return nil, fmt.Errorf("failed to execute credentials file template: %w", err)
	}

	return buff.Bytes(), nil
}
