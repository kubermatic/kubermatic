package resources

import (
	"bytes"
	"fmt"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/nodeportproxy"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilpointer "k8s.io/utils/pointer"
)

const (
	OauthName                 = "openshift-oauth"
	oauthCLIConfigTemplateRaw = `
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
          "sessionSecretsFile": "/var/config/system/secrets/v4-0-config-system-session/v4-0-config-system-session"
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
        "certFile": "/var/config/system/secrets/v4-0-config-system-serving-cert/tls.crt",
        "keyFile": "/var/config/system/secrets/v4-0-config-system-serving-cert/tls.key",
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
	oauthDeploymentResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("10m"),
			corev1.ResourceMemory: resource.MustParse("50Mi"),
		},
	}
	oauthCLIConfigTemplate = template.Must(template.New("base").Funcs(sprig.TxtFuncMap()).Parse(oauthCLIConfigTemplateRaw))
)

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

// OauthServiceCreator returns the function to reconcile the external OpenVPN service
func OauthServiceCreator(exposeStrategy corev1.ServiceType) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return OauthName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Labels = resources.BaseAppLabel(name, nil)

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

func OauthDeploymentCreator(data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return OauthName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {

			dep.Spec.Replicas = utilpointer.Int32Ptr(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(OauthName, nil),
			}
			dep.Spec.Template.Labels = resources.BaseAppLabel(OauthName, nil)
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: resources.ImagePullSecretName},
				{Name: openshiftImagePullSecretName},
			}
			dep.Spec.Template.Spec.AutomountServiceAccountToken = utilpointer.BoolPtr(false)
			image, err := getOauthImage(data.Cluster().Spec.Version.String())
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
			}
			dep.Spec.Template.Spec.Containers = []corev1.Container{{
				Name:  OauthName,
				Image: image,
				Command: []string{
					"hypershift",
					"openshift-osinserver",
					"--config=/etc/oauth/config.yaml",
					"--v=2",
				},
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
				Resources: *oauthDeploymentResourceRequirements.DeepCopy(),
			}}

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(OpenshiftAPIServerDeploymentName, data.Cluster().Name)
			podLabels, err := data.GetPodTemplateLabels(OauthName, dep.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}
			dep.Spec.Template.Labels = podLabels
			return dep, nil
		}
	}
}

func getOauthImage(openshiftVersion string) (string, error) {
	switch openshiftVersion {
	case openshiftVersion419:
		return "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:86255c4efe6bbc141a0f41444f863bbd5cd832ffca21d2b737a4f9c225ed00ad", nil
	default:
		return "", fmt.Errorf("no image for openshift version %q", openshiftVersion)
	}
}
