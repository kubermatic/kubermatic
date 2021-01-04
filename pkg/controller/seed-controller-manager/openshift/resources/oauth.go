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
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/servingcerthelper"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
)

const (
	OauthName = "openshift-oauth"
	// OAuthServiceName is the name of the OAuthService
	OAuthServiceName           = OauthName
	oauthSessionSecretName     = "openshift-oauth-session-secret"
	oauthServingCertSecretName = "openshift-oauth-serving-cert"
	oauthCLIConfigTemplateRaw  = `
    {
      "admission": {},
      "apiVersion": "osin.config.openshift.io/v1",
      "auditConfig": {
        "auditFilePath": "",
        "enabled": false,
        "logFormat": "",
        "maximumFileRetentionDays": 0,
        "maximumFileSizeMegabytes": 0,
        "maximumRetainedFiles": 0,
        "policyConfiguration": null,
        "policyFile": "",
        "webHookKubeConfig": "",
        "webHookMode": ""
      },
      "corsAllowedOrigins": null,
      "kind": "OsinServerConfig",
      "kubeClientConfig": {
        "connectionOverrides": {
          "acceptContentTypes": "",
          "burst": 400,
          "contentType": "",
          "qps": 400
        },
        "kubeConfig": "/etc/kubernetes/kubeconfig/kubeconfig"
      },
      "oauthConfig": {
        "alwaysShowProviderSelection": false,
        "assetPublicURL": "https://console-openshift-console.apps.alvaro-test.aws.k8c.io",
        "grantConfig": {
          "method": "deny",
          "serviceAccountMethod": "prompt"
        },
        "identityProviders": [],
        "loginURL": "{{ .APIServerURL }}",
        "masterCA": "/var/config/system/configmaps/v4-0-config-system-service-ca/service-ca.crt",
        "masterPublicURL": "https://{{ .ExternalName }}:{{ .OauthPort }}",
        "masterURL": "https://openshift-oauth.{{ .InternalURL }}",
        "sessionConfig": {
          "sessionMaxAgeSeconds": 300,
          "sessionName": "ssn",
          "sessionSecretsFile": "/etc/openshift-oauth-session-secret/openshift-oauth-session-secret"
        },
        "templates": {
          "error": "/var/config/system/secrets/v4-0-config-system-ocp-branding-template/errors.html",
          "login": "/var/config/system/secrets/v4-0-config-system-ocp-branding-template/login.html",
          "providerSelection": "/var/config/system/secrets/v4-0-config-system-ocp-branding-template/providers.html"
        },
        "tokenConfig": {
          "accessTokenMaxAgeSeconds": 86400,
          "authorizeTokenMaxAgeSeconds": 300
        }
      },
      "servingInfo": {
        "bindAddress": "0.0.0.0:6443",
        "bindNetwork": "tcp4",
        "cipherSuites": [
          "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
          "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
          "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
          "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
          "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
          "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
          "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
          "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
          "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
          "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
          "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
          "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
          "TLS_RSA_WITH_AES_128_GCM_SHA256",
          "TLS_RSA_WITH_AES_256_GCM_SHA384",
          "TLS_RSA_WITH_AES_128_CBC_SHA",
          "TLS_RSA_WITH_AES_256_CBC_SHA"
        ],
        "certFile": "/etc/servingcert/serving.crt",
        "keyFile": "/etc/servingcert/serving.key",
        "maxRequestsInFlight": 1000,
        "minTLSVersion": "VersionTLS12",
        "namedCertificates": [],
        "requestTimeoutSeconds": 300
      },
      "storageConfig": {
        "ca": "",
        "certFile": "",
        "keyFile": "",
        "storagePrefix": ""
      }
    }
`
)

var (
	oauthDeploymentResourceRequirements = map[string]*corev1.ResourceRequirements{
		OauthName: {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
		},
	}
	oauthCLIConfigTemplate = template.Must(template.New("base").Funcs(sprig.TxtFuncMap()).Parse(oauthCLIConfigTemplateRaw))
)

func OauthTLSServingCertCreator(data openshiftData) reconciling.NamedSecretCreatorGetter {
	return servingcerthelper.ServingCertSecretCreator(data.GetRootCA,
		oauthServingCertSecretName,
		data.Cluster().Address.ExternalName,
		[]string{data.Cluster().Address.ExternalName},
		nil)
}

func OauthConfigMapCreator(data openshiftData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return OauthName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			oauthPort, err := data.GetOauthExternalNodePort()
			if err != nil {
				return nil, fmt.Errorf("failed to get port for oauth service: %v", err)
			}
			templateData := struct {
				ExternalName string
				OauthPort    string
				APIServerURL string
				InternalURL  string
			}{
				ExternalName: data.Cluster().Address.ExternalName,
				OauthPort:    strconv.Itoa(int(oauthPort)),
				APIServerURL: data.Cluster().Address.URL,
				InternalURL:  data.Cluster().Address.InternalName,
			}
			oauthConfig := bytes.NewBuffer([]byte{})
			if err := oauthCLIConfigTemplate.Execute(oauthConfig, templateData); err != nil {
				return nil, fmt.Errorf("failed to render oauth config template: %v", err)
			}

			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Data["config.yaml"] = oauthConfig.String()
			return cm, nil
		}
	}
}

// OauthServiceCreator returns the function to reconcile the external Oauth service
func OauthServiceCreator(exposeStrategy corev1.ServiceType) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return OAuthServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Labels = resources.BaseAppLabels(name, nil)

			if se.Annotations == nil {
				se.Annotations = map[string]string{}
			}
			if exposeStrategy == corev1.ServiceTypeNodePort {
				se.Annotations["nodeport-proxy.k8s.io/expose"] = "true"
				delete(se.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			} else {
				se.Annotations[nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey] = "true"
				delete(se.Annotations, "nodeport-proxy.k8s.io/expose")
			}
			se.Spec.Selector = map[string]string{
				resources.AppLabelKey: OauthName,
			}
			se.Spec.Type = corev1.ServiceTypeNodePort
			if len(se.Spec.Ports) == 0 {
				se.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			se.Spec.Ports[0].Name = OauthName
			se.Spec.Ports[0].Port = 443
			se.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			se.Spec.Ports[0].TargetPort = intstr.FromInt(6443)

			return se, nil
		}
	}
}

func OauthSessionSecretCreator() (string, reconciling.SecretCreator) {
	return oauthSessionSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
		if s.Data == nil {
			s.Data = map[string][]byte{}
		}
		if s.Data[oauthSessionSecretName] == nil {
			skey, err := newSessionSecretsJSON()
			if err != nil {
				return nil, fmt.Errorf("failed to generate sessionSecret: %v", err)
			}
			s.Data[oauthSessionSecretName] = skey
		}
		return s, nil
	}
}

// Copied code start
//
// SessionSecrets list the secrets to use to sign/encrypt and authenticate/decrypt created sessions.
type SessionSecrets struct {
	metav1.TypeMeta `json:",inline"`

	// Secrets is a list of secrets
	// New sessions are signed and encrypted using the first secret.
	// Existing sessions are decrypted/authenticated by each secret until one succeeds. This allows rotating secrets.
	Secrets []SessionSecret `json:"secrets"`
}

// SessionSecret is a secret used to authenticate/decrypt cookie-based sessions
type SessionSecret struct {
	// Authentication is used to authenticate sessions using HMAC. Recommended to use a secret with 32 or 64 bytes.
	Authentication string `json:"authentication"`
	// Encryption is used to encrypt sessions. Must be 16, 24, or 32 characters long, to select AES-128, AES-
	Encryption string `json:"encryption"`
}

// Straight copied from upstream at
// https://github.com/openshift/cluster-authentication-operator/blob/21ada6ef0fe098e4b6ca67096b3f146c04be0b77/pkg/operator2/secret.go#L70
func newSessionSecretsJSON() ([]byte, error) {
	const (
		sha256KeyLenBytes = sha256.BlockSize // max key size with HMAC SHA256
		aes256KeyLenBytes = 32               // max key size with AES (AES-256)
	)

	secrets := &SessionSecrets{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SessionSecrets",
			APIVersion: "v1",
		},
		Secrets: []SessionSecret{
			{
				Authentication: randomString(sha256KeyLenBytes), // 64 chars
				Encryption:     randomString(aes256KeyLenBytes), // 32 chars
			},
		},
	}
	secretsBytes, err := json.Marshal(secrets)
	if err != nil {
		return nil, fmt.Errorf("error marshalling the session secret: %v", err) // should never happen
	}

	return secretsBytes, nil
}

// randomString uses RawURLEncoding to ensure we do not get / characters or trailing ='s
func randomString(size int) string {
	// each byte (8 bits) gives us 4/3 base64 (6 bits) characters
	// we account for that conversion and add one to handle truncation
	b64size := base64.RawURLEncoding.DecodedLen(size) + 1
	// trim down to the original requested size since we added one above
	return base64.RawURLEncoding.EncodeToString(randomBytes(b64size))[:size]
}

// needs to be in lib-go
func randomBytes(size int) []byte {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		panic(err) // rand should never fail
	}
	return b
}

//
// Copied code end

func OauthDeploymentCreator(data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return OauthName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {

			dep.Spec.Replicas = utilpointer.Int32Ptr(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(OauthName, nil),
			}
			dep.Spec.Template.Labels = resources.BaseAppLabels(OauthName, nil)
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: resources.ImagePullSecretName},
				{Name: openshiftImagePullSecretName},
			}
			dep.Spec.Template.Spec.AutomountServiceAccountToken = utilpointer.BoolPtr(false)
			image, err := hypershiftImage(data.Cluster().Spec.Version.String(), data.ImageRegistry(""))
			if err != nil {
				return nil, err
			}
			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: OauthName},
						},
					},
				},
				{
					Name: resources.InternalUserClusterAdminKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
						},
					},
				},
				{
					Name: oauthSessionSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: oauthSessionSecretName,
						},
					},
				},
				{
					Name: oauthOCPBrandingSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: oauthOCPBrandingSecretName,
						},
					},
				},
				{
					Name: oauthServingCertSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: oauthServingCertSecretName,
						},
					},
				},
			}

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{{
				Name:  OauthName,
				Image: image,
				Command: []string{
					"hypershift",
					"openshift-osinserver",
					"--config=/etc/oauth/config.yaml",
					"--v=2",
				},
				Env: []corev1.EnvVar{{
					Name:  "KUBECONFIG",
					Value: "/etc/kubernetes/kubeconfig/kubeconfig",
				}},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "config",
						MountPath: "/etc/oauth",
					},
					{
						Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
						MountPath: "/etc/kubernetes/kubeconfig",
						ReadOnly:  true,
					},
					{
						Name:      oauthSessionSecretName,
						MountPath: "/etc/" + oauthSessionSecretName,
					},
					{
						Name:      oauthOCPBrandingSecretName,
						MountPath: "/var/config/system/secrets/v4-0-config-system-ocp-branding-template",
					},
					{
						Name:      oauthServingCertSecretName,
						MountPath: "/etc/servingcert",
					},
				},
				LivenessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						HTTPGet: &corev1.HTTPGetAction{
							Path:   "/healthz",
							Port:   intstr.FromInt(6443),
							Scheme: "HTTPS",
						},
					},
					FailureThreshold:    3,
					PeriodSeconds:       10,
					SuccessThreshold:    1,
					TimeoutSeconds:      1,
					InitialDelaySeconds: 30,
				},
				ReadinessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						HTTPGet: &corev1.HTTPGetAction{
							Path:   "/healthz",
							Port:   intstr.FromInt(6443),
							Scheme: "HTTPS",
						},
					},
					FailureThreshold: 3,
					PeriodSeconds:    10,
					SuccessThreshold: 1,
					TimeoutSeconds:   1,
				},
			}}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, oauthDeploymentResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}
			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(OpenshiftAPIServerDeploymentName, data.Cluster().Name)
			podLabels, err := data.GetPodTemplateLabels(OauthName, dep.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}
			dep.Spec.Template.Labels = podLabels

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(OauthName), "OAuthClient,oauth.openshift.io/v1")
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec
			return dep, nil
		}
	}
}
