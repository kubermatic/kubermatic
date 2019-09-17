package resources

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"text/template"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/servingcerthelper"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
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
	consoleDeploymentName        = "openshift-console"
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
  certFile: /var/serving-cert/serving.crt
  keyFile: /var/serving-cert/serving.key
`
)

func ConsoleDeployment(data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return consoleDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {

			d.Spec.Template.Spec.AutomountServiceAccountToken = utilpointer.BoolPtr(false)
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: openshiftImagePullSecretName}}
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(consoleDeploymentName, nil),
			}
			image, err := getConsoleImage(data.Cluster().Spec.Version.String())
			if err != nil {
				return nil, err
			}

			d.Spec.Template.Spec.Containers = []corev1.Container{{
				Name:  "console",
				Image: image,
				Command: []string{
					"/opt/bridge/bin/bridge",
					"--public-dir=/opt/bridge/static",
					"--config=/etc/console-config/console-config.yaml",
				},
				Env: []corev1.EnvVar{
					{
						Name:  "KUBERNETES_SERVICE_HOST",
						Value: data.Cluster().Address.InternalName,
					},
					{
						Name:  "KUBERNETES_SERVICE_PORT",
						Value: strconv.Itoa(int(data.Cluster().Address.Port)),
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      consoleOauthSecretName,
						MountPath: "/var/oauth-config",
					},
					{
						Name:      consoleConfigMapName,
						MountPath: "/etc/console-config",
					},
					{
						Name:      consoleServingCertSecretName,
						MountPath: "/var/serving-cert",
					},
					{
						Name:      resources.CASecretName,
						MountPath: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
						SubPath:   "ca.crt",
					},
					{
						Name:      resources.AdminKubeconfigSecretName,
						MountPath: "/var/run/secrets/kubernetes.io/serviceaccount/token",
						SubPath:   "token",
					},
				},
			}}
			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: resources.AdminKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{SecretName: resources.AdminKubeconfigSecretName},
					},
				},
				{
					Name: consoleOauthSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{SecretName: consoleOauthSecretName},
					},
				},
				{
					Name: consoleServingCertSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{SecretName: consoleServingCertSecretName},
					},
				},
				{
					Name: consoleConfigMapName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: consoleConfigMapName},
						},
					},
				},
				{
					Name: resources.CASecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.CASecretName,
						},
					},
				},
			}

			podLabels, err := data.GetPodTemplateLabels(consoleDeploymentName, d.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}
			d.Spec.Template.Labels = podLabels

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, d.Spec.Template.Spec, sets.NewString("console"))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			d.Spec.Template.Spec = *wrappedPodSpec
			return d, nil
		}
	}
}

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

func getConsoleImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case "4.1.9":
		return "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:9e554ac4505edd925eb73fec52e33d7418e2cfaf8058b59d8246ed478337748d", nil
	default:
		return "", fmt.Errorf("no openshhift console image available for version %q", openshiftVersion)
	}
}
