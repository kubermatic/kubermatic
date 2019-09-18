package resources

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
)

var (
	kubeControllerManagerConfigTemplate = template.Must(template.New("base").Funcs(sprig.TxtFuncMap()).Parse(kubeControllerManagerConfigTemplateRaw))
)

const (
	kubeControllerManagerOpenshiftConfigConfigmapName = "openshift-kube-controller-manager-config"
	kubeControllerManagerOpenshiftConfigConfigMapKey  = "config.yaml"
	kubeControllerManagerConfigTemplateRaw            = `
apiVersion: kubecontrolplane.config.openshift.io/v1
kind: KubeControllerManagerConfig
serviceServingCert:
  certFile: {{ .CACertPath }}
extendedArguments:
  allocate-node-cidrs:
  - 'true'
  cert-dir:
  - /var/run/kubernetes
  # Must be an explicit empty slice, else the controller-manager panics
  cloud-provider: []
  #- aws
  cluster-cidr:
  - {{ .ClusterCIDR }}
  cluster-signing-cert-file:
  - {{ .CACertPath }}
  cluster-signing-key-file:
  - {{ .CAKeyPath }}
  configure-cloud-routes:
  - 'false'
  controllers:
  - '*'
  - -ttl
  - -bootstrapsigner
  - -tokencleaner
  enable-dynamic-provisioning:
  - 'true'
  experimental-cluster-signing-duration:
  - 720h
  feature-gates:
  - ExperimentalCriticalPodAnnotation=true
  - RotateKubeletServerCertificate=true
  - SupportPodPidsLimit=true
  - LocalStorageCapacityIsolation=false
  flex-volume-plugin-dir:
  - /etc/kubernetes/kubelet-plugins/volume/exec
  kube-api-burst:
  - '300'
  kube-api-qps:
  - '150'
  leader-elect:
  - 'true'
  leader-elect-resource-lock:
  # - configmaps
  # For some reason updating results in a 403 and upstream Openshift doesn't have extra bindings either?
  - endpoints
  leader-elect-retry-period:
  - 3s
  port:
  - '0'
  root-ca-file:
  - {{ .CACertPath }}
  secure-port:
  - '10257'
  service-account-private-key-file:
  - {{ .ServiceAccountKeyFile }}
  service-cluster-ip-range:
  - {{ .ServiceCIDR }}
  use-service-account-credentials:
  - 'true'
`
	kubeControllerManagerCACertPath            = "/etc/kubernetes/pki/ca/ca.crt"
	kubeControllerManagerCAKeyPath             = "/etc/kubernetes/pki/ca/ca.key"
	kubeControllerManagerServiceAccountKeyPath = "/etc/kubernetes/service-account-key/sa.key"
)

var (
	defaultKubeControllerResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("100Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2Gi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}
)

type kubeControllerManagerConfigData interface {
	Cluster() *kubermaticv1.Cluster
}

func KubeControllerManagerConfigMapCreatorFactory(data kubeControllerManagerConfigData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return kubeControllerManagerOpenshiftConfigConfigmapName,
			func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {

				if cm.Data == nil {
					cm.Data = map[string]string{}
				}

				var podCIDR, serviceCIDR string
				if len(data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks) > 0 {
					podCIDR = data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks[0]
				}
				if len(data.Cluster().Spec.ClusterNetwork.Services.CIDRBlocks) > 0 {
					serviceCIDR = data.Cluster().Spec.ClusterNetwork.Services.CIDRBlocks[0]
				}
				vars := struct {
					CACertPath            string
					CAKeyPath             string
					ClusterCIDR           string
					ServiceAccountKeyFile string
					ServiceCIDR           string
				}{
					CACertPath:            kubeControllerManagerCACertPath,
					CAKeyPath:             kubeControllerManagerCAKeyPath,
					ClusterCIDR:           podCIDR,
					ServiceAccountKeyFile: kubeControllerManagerServiceAccountKeyPath,
					ServiceCIDR:           serviceCIDR,
				}

				templateBuffer := &bytes.Buffer{}
				if err := kubeControllerManagerConfigTemplate.Execute(templateBuffer, vars); err != nil {
					return nil, fmt.Errorf("failed to execute template: %v", err)
				}
				cm.Data[kubeControllerManagerOpenshiftConfigConfigMapKey] = templateBuffer.String()

				return cm, nil
			}
	}
}

type kubeControllerManagerData interface {
	Cluster() *kubermaticv1.Cluster
	ClusterIPByServiceName(name string) (string, error)
	ImageRegistry(defaultRegistry string) string
	GetPodTemplateLabels(appName string, volumes []corev1.Volume, additionalLabels map[string]string) (map[string]string, error)
}

func KubeControllerManagerDeploymentCreatorFactory(data kubeControllerManagerData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.ControllerManagerDeploymentName,
			func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {

				dep.Spec.Replicas = utilpointer.Int32Ptr(1)
				if data.Cluster().Spec.ComponentsOverride.ControllerManager.Replicas != nil {
					dep.Spec.Replicas = data.Cluster().Spec.ComponentsOverride.ControllerManager.Replicas
				}
				dep.Spec.Selector = &metav1.LabelSelector{
					MatchLabels: resources.BaseAppLabel(resources.ControllerManagerDeploymentName, nil),
				}
				dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
					{Name: openshiftImagePullSecretName},
				}
				dep.Spec.Template.Spec.Volumes = kubeControllerManagerVolumes()

				image, err := kubeControllerManagerImage(data.Cluster().Spec.Version.String())
				if err != nil {
					return nil, err
				}

				dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
				if err != nil {
					return nil, err
				}

				resourceRequirements := defaultKubeControllerResourceRequirements
				if data.Cluster().Spec.ComponentsOverride.ControllerManager.Resources != nil {
					resourceRequirements = *data.Cluster().Spec.ComponentsOverride.ControllerManager.Resources
				}

				openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
				if err != nil {
					return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
				}
				kubeControllerManagerContainerName := "kube-controller-manager"
				dep.Spec.Template.Spec.Containers = []corev1.Container{
					*openvpnSidecar,
					{
						Name:    "kube-controller-manager",
						Image:   image,
						Command: []string{"hyperkube", kubeControllerManagerContainerName},
						Args: kubeControllerManagerArgs(
							"/etc/origin/config.yaml",
							"/etc/kubernetes/kubeconfig/kubeconfig",
							kubeControllerManagerCACertPath,
							"/etc/kubernetes/pki/front-proxy/ca/ca.crt",
						),
						Resources: resourceRequirements,
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "healthz",
									Scheme: corev1.URISchemeHTTPS,
									Port:   intstr.FromInt(10257),
								},
							},
							FailureThreshold: 3,
							PeriodSeconds:    10,
							SuccessThreshold: 1,
							TimeoutSeconds:   15,
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      resources.CASecretName,
								MountPath: "/etc/kubernetes/pki/ca",
								ReadOnly:  true,
							},
							{
								Name:      resources.ServiceAccountKeySecretName,
								MountPath: "/etc/kubernetes/service-account-key",
								ReadOnly:  true,
							},
							{
								Name:      resources.ControllerManagerKubeconfigSecretName,
								MountPath: "/etc/kubernetes/kubeconfig",
								ReadOnly:  true,
							},
							{
								Name:      kubeControllerManagerOpenshiftConfigConfigmapName,
								MountPath: "/etc/origin",
								ReadOnly:  true,
							},
							{
								Name:      resources.FrontProxyCASecretName,
								MountPath: "/etc/kubernetes/pki/front-proxy/ca",
								ReadOnly:  true,
							},
						},
					},
				}

				podLabels, err := data.GetPodTemplateLabels(resources.ControllerManagerDeploymentName, dep.Spec.Template.Spec.Volumes, nil)
				if err != nil {
					return nil, err
				}
				dep.Spec.Template.Labels = podLabels
				wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(kubeControllerManagerContainerName), "", "")
				if err != nil {
					return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
				}
				dep.Spec.Template.Spec = *wrappedPodSpec

				return dep, nil
			}
	}
}

func kubeControllerManagerImage(version string) (string, error) {
	switch version {
	case openshiftVersion419:
		return "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:155ef40a64608c946ca9ca0310bbf88f5a4664b2925502b3acac86847bc158e6", nil
	default:
		return "", fmt.Errorf("no kube-controller-image available for version %q", version)
	}
}

func kubeControllerManagerArgs(openshiftConfigPath, kubeconfigPath, caCertPath, aggregatorCACertPath string) []string {
	return []string{
		fmt.Sprintf("--openshift-config=%s", openshiftConfigPath),
		fmt.Sprintf("--kubeconfig=%s", kubeconfigPath),
		fmt.Sprintf("--authentication-kubeconfig=%s", kubeconfigPath),
		fmt.Sprintf("--authorization-kubeconfig=%s", kubeconfigPath),
		fmt.Sprintf("--client-ca-file=%s", caCertPath),
		fmt.Sprintf("--requestheader-client-ca-file=%s", aggregatorCACertPath),
		fmt.Sprint("-v=2"),
		// Used for metrics only, we can use a self-signed cert for now
		//fmt.Sprintf("--tls-cert-file=%s", servingCert),
		//fmt.Sprintf("--tls-private-key=%s", servingKey)
	}
}

func kubeControllerManagerVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.CASecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.ServiceAccountKeySecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.ServiceAccountKeySecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.OpenVPNClientCertificatesSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.ControllerManagerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.ControllerManagerKubeconfigSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: kubeControllerManagerOpenshiftConfigConfigmapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: kubeControllerManagerOpenshiftConfigConfigmapName,
					},
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.FrontProxyCASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.FrontProxyCASecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
	}
}
