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

package resources

import (
	"bytes"
	"fmt"
	"text/template"

	"k8c.io/kubermatic/v2/pkg/ee/cluster-backup/master/storage-location-controller/backupstore/aws"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// SecretReconciler returns a function to create the Secret containing the backup destination credentials.
func SecretReconciler(credentials *corev1.Secret) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return CloudCredentialsSecretName, func(cm *corev1.Secret) (*corev1.Secret, error) {
			awsAccessKeyID := credentials.Data[aws.AccessKeyIDKeyName]
			awsSecretAccessKey := credentials.Data[aws.SecretAccessKeyName]
			if awsAccessKeyID == nil || awsSecretAccessKey == nil {
				return nil, fmt.Errorf("backup destination credentials secret is not set correctly: [%s] and [%s] can't be empty", aws.AccessKeyIDKeyName, aws.SecretAccessKeyName)
			}

			cloudCredsFile, err := getVeleroCloudCredentials(awsAccessKeyID, awsSecretAccessKey)
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
aws_access_key_id = {{ .awsAccessKeyID }}
aws_secret_access_key = {{ .awsSecretAccessKey }}
`

func getVeleroCloudCredentials(awsAccessKeyID, awsSecretAccessKey []byte) ([]byte, error) {
	t, err := template.New("cloud-credentials").Parse(credentialsTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials file template: %w", err)
	}
	var buff bytes.Buffer
	if err := t.Execute(&buff, map[string]interface{}{
		"awsAccessKeyID":     string(awsAccessKeyID),
		"awsSecretAccessKey": string(awsSecretAccessKey),
	}); err != nil {
		return nil, fmt.Errorf("failed to execute credentials file template: %w", err)
	}

	return buff.Bytes(), nil
}
