package resources

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"text/template"

	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/servingcerthelper"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

var (
	consoleTemplate = template.Must(template.New("base").Parse(consoleTemplateRaw))
)

const (
	consoleOauthSecretName       = "openshift-console-oauth-client-secret"
	consoleServingCertSecretName = "openshift-console-serving-cert"
	consoleOauthClientObjectName = "console"
	consoleConfigMapName         = "openshift-console-config"
	consoleConfigMapKey          = "console-config.yaml"
	consoleTemplateRaw           = `apiVersion: console.openshift.io/v1
auth:
  clientID: console
  clientSecretFile: /var/oauth-config/clientSecret
  logoutRedirect: ""
  oauthEndpointCAFile: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
clusterInfo:
  consoleBaseAddress: https://{{ .ExternalName }}:443
  consoleBasePath: ""
  masterPublicURL: {{ .APIServerURL}}
customization:
  branding: ocp
  documentationBaseURL: https://docs.openshift.com/container-platform/4.1/
kind: ConsoleConfig
servingInfo:
  bindAddress: https://0.0.0.0:8443
  certFile: /var/serving-cert/tls.crt
  keyFile: /var/serving-cert/tls.key
`
)

func ConsoleConfigCreator(data openshiftData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return consoleConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {

			data := struct {
				APIServerURL string
				ExternalName string
			}{
				APIServerURL: data.Cluster().Address.URL,
				ExternalName: data.Cluster().Address.ExternalName,
			}
			buffer := bytes.NewBuffer([]byte{})
			if err := consoleTemplate.Execute(buffer, data); err != nil {
				return nil, fmt.Errorf("failed to render template for openshift console: %v", err)
			}

			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Data[consoleConfigMapKey] = buffer.String()
			return cm, nil
		}
	}
}

func ConsoleServingCertCreator(caGetter servingcerthelper.CAGetter) reconciling.NamedSecretCreatorGetter {
	return servingcerthelper.ServingCertSecretCreator(caGetter,
		consoleServingCertSecretName,
		// We proxy this from the API
		"console.openshift.seed.tld",
		[]string{"console.openshift.seed.tld"},
		nil)
}

func ConsoleOauthClientSecretCreator(data openshiftData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return consoleOauthSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			oauthClientObject := &unstructured.Unstructured{}
			oauthClientObject.SetAPIVersion("oauth.openshift.io/v1")
			oauthClientObject.SetKind("OAuthClient")

			client, err := data.Client()
			if err != nil {
				return nil, fmt.Errorf("failed to get usercluster client: %v", err)
			}

			// Create oauthClient object in the usercluster first, as it can not be reset otherwise
			// because end-users do not have access to the seed
			name := types.NamespacedName{Name: consoleOauthClientObjectName}
			if err := client.Get(context.Background(), name, oauthClientObject); err != nil {
				if !kerrors.IsNotFound(err) {
					return nil, fmt.Errorf("failed to get OauthClient %q from usercluster: %v", consoleOauthClientObjectName, err)
				}
				secret, err := generateNewSecret()
				if err != nil {
					return nil, fmt.Errorf("failed to generate oauth client secret: %v", err)
				}
				if oauthClientObject.Object == nil {
					oauthClientObject.Object = map[string]interface{}{}
				}
				oauthClientObject.Object["secret"] = secret
				oauthClientObject.Object["redirectURIs"] = []string{
					// TODO: Insert something proper
					"https://console-openshift-console.apps.alvaro-test.aws.k8c.io/auth/callback",
				}
				oauthClientObject.Object["grantMethod"] = "auto"
				oauthClientObject.SetName(consoleOauthClientObjectName)
				if err := client.Create(context.Background(), oauthClientObject); err != nil {
					return nil, fmt.Errorf("failed to create OauthClient object in user cluster: %v", err)
				}
			}

			stringVal, ok := oauthClientObject.Object["secret"].(string)
			if !ok {
				return nil, fmt.Errorf("`secret` field of OAuthClient object was not a string but a %T", oauthClientObject.Object["secret"])
			}

			if s.Data == nil {
				s.Data = map[string][]byte{}
			}
			s.Data["clientSecret"] = []byte(stringVal)

			return s, nil
		}
	}
}

func generateNewSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read from crypto/rand: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
