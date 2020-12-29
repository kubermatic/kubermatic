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
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig"
	"k8c.io/kubermatic/v2/pkg/resources/controllermanager"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"

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
{{- if .CloudProvider }}
  cloud-provider:
  - {{ .CloudProvider}}
  cloud-config:
  - /etc/kubernetes/cloud/config
{{- end }}
  cluster-cidr:
  - {{ .ClusterCIDR }}
  cluster-signing-cert-file:
  - {{ .CACertPath }}
  cluster-signing-key-file:
  - {{ .CAKeyPath }}
{{- if .ConfigureCloudRoutes }}
  configure-cloud-routes:
  - '{{ .ConfigureCloudRoutes }}'
{{- end }}
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
	GetKubernetesCloudProviderName() string
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
				configureCloudRoutes := ""
				configureCloudRoutesBoolPtr := controllermanager.CloudRoutesFlagVal(data.Cluster().Spec.Cloud)
				if configureCloudRoutesBoolPtr != nil {
					configureCloudRoutes = fmt.Sprintf("%t", *configureCloudRoutesBoolPtr)
				}
				vars := struct {
					CACertPath            string
					CAKeyPath             string
					ClusterCIDR           string
					ServiceAccountKeyFile string
					ServiceCIDR           string
					CloudProvider         string
					ConfigureCloudRoutes  string
				}{
					CACertPath:            kubeControllerManagerCACertPath,
					CAKeyPath:             kubeControllerManagerCAKeyPath,
					ClusterCIDR:           podCIDR,
					ServiceAccountKeyFile: kubeControllerManagerServiceAccountKeyPath,
					ServiceCIDR:           serviceCIDR,
					CloudProvider:         data.GetKubernetesCloudProviderName(),
					ConfigureCloudRoutes:  configureCloudRoutes,
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
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
	Seed() *kubermaticv1.Seed
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
					MatchLabels: resources.BaseAppLabels(resources.ControllerManagerDeploymentName, nil),
				}
				dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
					{Name: openshiftImagePullSecretName},
				}
				dep.Spec.Template.Spec.Volumes = kubeControllerManagerVolumes()
				volumeMounts := []corev1.VolumeMount{
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
					{
						Name:      resources.CloudConfigConfigMapName,
						MountPath: "/etc/kubernetes/cloud",
						ReadOnly:  true,
					},
				}

				if data.Cluster().Spec.Cloud.VSphere != nil {
					fakeVMWareUUIDMount := corev1.VolumeMount{
						Name:      resources.CloudConfigConfigMapName,
						SubPath:   cloudconfig.FakeVMWareUUIDKeyName,
						MountPath: "/sys/class/dmi/id/product_serial",
						ReadOnly:  true,
					}
					// Required because of https://github.com/kubernetes/kubernetes/issues/65145
					volumeMounts = append(volumeMounts, fakeVMWareUUIDMount)
				}

				image, err := hyperkubeImage(data.Cluster().Spec.Version.String(), data.ImageRegistry(""))
				if err != nil {
					return nil, err
				}

				dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
				if err != nil {
					return nil, err
				}

				env, err := controllermanager.GetEnvVars(data)
				if err != nil {
					return nil, fmt.Errorf("failed to get controller manager env vars: %v", err)
				}

				openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
				if err != nil {
					return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
				}
				kubeControllerManagerContainerName := "kube-controller-manager"

				dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
				dep.Spec.Template.Spec.Containers = []corev1.Container{
					*openvpnSidecar,
					{
						Name:    resources.ControllerManagerDeploymentName,
						Image:   image,
						Env:     env,
						Command: []string{"hyperkube", kubeControllerManagerContainerName},
						Args: kubeControllerManagerArgs(
							"/etc/origin/config.yaml",
							"/etc/kubernetes/kubeconfig/kubeconfig",
							kubeControllerManagerCACertPath,
							"/etc/kubernetes/pki/front-proxy/ca/ca.crt",
						),
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
						VolumeMounts: volumeMounts,
					},
				}
				defResourceRequirements := map[string]*corev1.ResourceRequirements{
					kubeControllerManagerContainerName: defaultKubeControllerResourceRequirements.DeepCopy(),
					openvpnSidecar.Name:                openvpnSidecar.Resources.DeepCopy(),
				}
				err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), dep.Annotations)
				if err != nil {
					return nil, fmt.Errorf("failed to set resource requirements: %v", err)
				}
				podLabels, err := data.GetPodTemplateLabels(resources.ControllerManagerDeploymentName, dep.Spec.Template.Spec.Volumes, nil)
				if err != nil {
					return nil, fmt.Errorf("failed to add template labels: %v", err)
				}
				dep.Spec.Template.Labels = podLabels
				wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(resources.ControllerManagerDeploymentName))
				if err != nil {
					return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
				}
				dep.Spec.Template.Spec = *wrappedPodSpec

				return dep, nil
			}
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
		"-v=2",
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
		{
			Name: resources.CloudConfigConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.CloudConfigConfigMapName,
					},
				},
			},
		},
	}
}
