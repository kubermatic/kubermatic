package resources

import (
	"fmt"

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
	oauthName                 = "openshift-oauth"
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
        # TODO: Oauth cluster service
        "masterURL": "https://openshift-oauth.{{ .ClusterNamespace }}.svc.cluster.local.",
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
        # TODO: Re-Enable
        "namedCertificates": [],
          #          {
          #            "certFile": "/var/config/system/secrets/v4-0-config-system-router-certs/apps.alvaro-test.aws.k8c.io",
          #            "keyFile": "/var/config/system/secrets/v4-0-config-system-router-certs/apps.alvaro-test.aws.k8c.io",
          #            "names": [
          #              "*.apps.alvaro-test.aws.k8c.io"
          #            ]
          #          }
          #        ],
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

var oauthDeploymentResourceRequirements = corev1.ResourceRequirements{
	Requests: corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("10m"),
		corev1.ResourceMemory: resource.MustParse("50Mi"),
	},
}

// OauthServiceCreator returns the function to reconcile the external OpenVPN service
func OauthServiceCreator(exposeStrategy corev1.ServiceType) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return oauthName, func(se *corev1.Service) (*corev1.Service, error) {
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
				resources.AppLabelKey: oauthName,
			}
			se.Spec.Type = corev1.ServiceTypeNodePort
			if len(se.Spec.Ports) == 0 {
				se.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			se.Spec.Ports[0].Name = oauthName
			se.Spec.Ports[0].Port = 443
			se.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			se.Spec.Ports[0].TargetPort = intstr.FromInt(6443)

			return se, nil
		}
	}
}

func OauthDeploymentCreator(data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return oauthName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {

			dep.Spec.Replicas = utilpointer.Int32Ptr(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(oauthName, nil),
			}
			dep.Spec.Template.Labels = resources.BaseAppLabel(oauthName, nil)
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: resources.ImagePullSecretName},
				{Name: openshiftImagePullSecretName},
			}
			dep.Spec.Template.Spec.AutomountServiceAccountToken = utilpointer.BoolPtr(false)
			image, err := getOauthImage(data.Cluster().Spec.Version.String())
			if err != nil {
				return nil, err
			}
			dep.Spec.Template.Spec.Containers = []corev1.Container{{
				Name:  oauthName,
				Image: image,
				Command: []string{
					"hypershift",
					"openshift-osinserver",
					"--config=/var/config/system/configmaps/v4-0-config-system-cliconfig/v4-0-config-system-cliconfig",
					"--v=2",
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
