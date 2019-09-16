package resources

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

const (
	consoleOauthSecretName       = "console-oauth-client-secret"
	consoleOauthClientObjectName = "console"
)

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
