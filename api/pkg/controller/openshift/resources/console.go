package resources

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"strconv"
	"text/template"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
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
	consoleOAuthSecretName       = "openshift-console-oauth-client-secret"
	consoleServingCertSecretName = "openshift-console-serving-cert"
	consoleOAuthClientObjectName = "console"
	consoleConfigMapName         = "openshift-console-config"
	consoleConfigMapKey          = "console-config.yaml"
	consoleDeploymentName        = "openshift-console"
	// ConsoleAdminPasswordSecretName is the name of the secret that contains
	// the bootstrap admin user for Openshift OAuth
	ConsoleAdminPasswordSecretName = "openshift-bootstrap-password"
	// ConsoleAdminUserName is the name of the bootstrap admin user for oauth/the console
	ConsoleAdminUserName = "kubeadmin"
	// ConsoleListenPort is the port the console listens on
	ConsoleListenPort  = 8443
	consoleTemplateRaw = `apiVersion: console.openshift.io/v1
auth:
  clientID: console
  clientSecretFile: /var/oauth-config/clientSecret
  logoutRedirect: https://{{ .ExternalURL }}/api/v1/projects/{{ .ProjectID }}/dc/{{ .SeedName }}/clusters/{{ .ClusterName }}/openshift/console/login
  oauthEndpointCAFile: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
clusterInfo:
  consoleBaseAddress: https://{{ .ExternalURL }}
  consoleBasePath: /api/v1/projects/{{ .ProjectID }}/dc/{{ .SeedName }}/clusters/{{ .ClusterName }}/openshift/console/proxy/
  masterPublicURL: {{ .APIServerURL }}
customization:
  branding: ocp
  documentationBaseURL: https://docs.openshift.com/container-platform/4.1/
kind: ConsoleConfig
servingInfo:
  bindAddress: http://0.0.0.0:{{ .ListenPort }}
  certFile: /var/serving-cert/serving.crt
  keyFile: /var/serving-cert/serving.key`
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
					{
						Name:  "KUBECONFIG",
						Value: "/etc/kubernetes/kubeconfig/kubeconfig",
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      consoleOAuthSecretName,
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
					// Used by the http prober
					{
						Name:      resources.AdminKubeconfigSecretName,
						MountPath: "/etc/kubernetes/kubeconfig",
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
					Name: consoleOAuthSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{SecretName: consoleOAuthSecretName},
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

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, d.Spec.Template.Spec, sets.NewString("console"), "OAuthClient,oauth.openshift.io/v1")
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
				ExternalURL  string
				ProjectID    string
				SeedName     string
				ClusterName  string
				ListenPort   string
			}{
				APIServerURL: data.Cluster().Address.URL,
				ExternalURL:  data.ExternalURL(),
				ProjectID:    data.Cluster().Labels[kubermaticv1.ProjectIDLabelKey],
				SeedName:     data.SeedName(),
				ClusterName:  data.Cluster().Name,
				ListenPort:   strconv.Itoa(ConsoleListenPort),
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

func ConsoleOAuthClientSecretCreator(data openshiftData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return consoleOAuthSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			oauthClientObject := &unstructured.Unstructured{}
			oauthClientObject.SetAPIVersion("oauth.openshift.io/v1")
			oauthClientObject.SetKind("OAuthClient")

			client, err := data.Client()
			if err != nil {
				return nil, fmt.Errorf("failed to get usercluster client: %v", err)
			}

			// Create oauthClient object in the usercluster first, as it can not be reset otherwise
			// because end-users do not have access to the seed
			name := types.NamespacedName{Name: consoleOAuthClientObjectName}
			if err := client.Get(context.Background(), name, oauthClientObject); err != nil {
				if !kerrors.IsNotFound(err) {
					return nil, fmt.Errorf("failed to get OAuthClient %q from usercluster: %v", consoleOAuthClientObjectName, err)
				}
				secret, err := generateNewSecret()
				if err != nil {
					return nil, fmt.Errorf("failed to generate OAuthClient secret: %v", err)
				}
				if oauthClientObject.Object == nil {
					oauthClientObject.Object = map[string]interface{}{}
				}
				oauthClientObject.Object["secret"] = secret
				oauthClientObject.Object["redirectURIs"] = []string{
					fmt.Sprintf("https://%s/api/v1/projects/%s/dc/%s/clusters/%s/openshift/console/proxy/auth/callback",
						data.ExternalURL(), data.Cluster().Labels[kubermaticv1.ProjectIDLabelKey], data.SeedName(), data.Cluster().Name),
				}
				oauthClientObject.Object["grantMethod"] = "auto"
				oauthClientObject.SetName(consoleOAuthClientObjectName)
				if err := client.Create(context.Background(), oauthClientObject); err != nil {
					return nil, fmt.Errorf("failed to create OAuthClient object in user cluster: %v", err)
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

func BootStrapPasswordSecretGenerator(data openshiftData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return ConsoleAdminPasswordSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			// Check if secret inside usercluster exists. It is only valid if its creation tiemestmap
			// is < kube-system creation timestamp + 1h
			userClusterClient, err := data.Client()
			if err != nil {
				return nil, fmt.Errorf("failed to get usercluster client: %v", err)
			}

			var rawPassword string
			userClusterSecretName := types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: "kubeadmin"}
			userClusterSecret := &corev1.Secret{}
			if err := userClusterClient.Get(context.Background(), userClusterSecretName, userClusterSecret); err != nil {
				if !kerrors.IsNotFound(err) {
					return nil, fmt.Errorf("failed to get secret %q from usercluster: %v", userClusterSecretName.String(), err)
				}

				rawPassword, err = generateNewSecret()
				if err != nil {
					return nil, fmt.Errorf("failed to generate password: %v", err)
				}
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), 12)
				if err != nil {
					return nil, fmt.Errorf("failed to hash password: %v", err)
				}

				userClusterSecret.Namespace = metav1.NamespaceSystem
				userClusterSecret.Name = "kubeadmin"
				userClusterSecret.Data = map[string][]byte{"kubeadmin": hashedPassword}
				if err := userClusterClient.Create(context.Background(), userClusterSecret); err != nil {
					return nil, fmt.Errorf("failed to create password hash in usercluster: %v", err)
				}
			}

			// TODO: This needs reworking, we can not fix the seed secret if someone changes it
			if len(s.Data[ConsoleAdminUserName]) == 0 {
				s.Data = map[string][]byte{"kubeadmin": []byte(rawPassword)}
			}
			return s, nil
		}
	}
}
