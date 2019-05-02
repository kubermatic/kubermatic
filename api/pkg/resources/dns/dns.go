package dns

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("20Mi"),
			corev1.ResourceCPU:    resource.MustParse("5m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("128Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
	}
)

// ServiceCreator returns the function to reconcile the DNS service
func ServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.DNSResolverServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Name = resources.DNSResolverServiceName
			se.Spec.Selector = resources.BaseAppLabel(resources.DNSResolverDeploymentName, nil)
			se.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "dns",
					Protocol:   corev1.ProtocolUDP,
					Port:       int32(53),
					TargetPort: intstr.FromInt(53),
				},
			}

			return se, nil
		}
	}
}

type deploymentCreatorData interface {
	Cluster() *kubermaticv1.Cluster
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	ImageRegistry(string) string
}

// DeploymentCreator returns the function to create and update the DNS resolver deployment
func DeploymentCreator(data deploymentCreatorData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.DNSResolverDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.DNSResolverDeploymentName
			dep.Labels = resources.BaseAppLabel(resources.DNSResolverDeploymentName, nil)
			dep.Spec.Replicas = resources.Int32(2)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(resources.DNSResolverDeploymentName, nil),
			}
			dep.Spec.Strategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
			dep.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
				MaxSurge: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 1,
				},
				MaxUnavailable: &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 0,
				},
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			volumes := getVolumes()
			podLabels, err := data.GetPodTemplateLabels(resources.DNSResolverDeploymentName, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to get podlabels: %v", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{Labels: podLabels}
			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn sidecar for dns resolver: %v", err)
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				{
					Name:      resources.DNSResolverDeploymentName,
					Image:     data.ImageRegistry(resources.RegistryGCR) + "/google_containers/coredns:1.1.3",
					Args:      []string{"-conf", "/etc/coredns/Corefile"},
					Resources: defaultResourceRequirements,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.DNSResolverConfigMapName,
							MountPath: "/etc/coredns",
							ReadOnly:  true,
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health",
								Port:   intstr.FromInt(8080),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 2,
						FailureThreshold:    3,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
				},
			}

			dep.Spec.Template.Spec.Volumes = volumes

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(resources.DNSResolverDeploymentName, data.Cluster().Name)

			return dep, nil
		}
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.DNSResolverConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.DNSResolverConfigMapName,
					},
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
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.CASecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
					Items: []corev1.KeyToPath{
						{
							Path: resources.CACertSecretKey,
							Key:  resources.CACertSecretKey,
						},
					},
				},
			},
		},
	}
}

type configMapCreatorData interface {
	Cluster() *kubermaticv1.Cluster
}

// ConfigMapCreator returns a ConfigMap containing the cloud-config for the supplied data
func ConfigMapCreator(data configMapCreatorData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.DNSResolverConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			dnsIP, err := resources.UserClusterDNSResolverIP(data.Cluster())
			if err != nil {
				return nil, err
			}
			seedClusterNamespaceDNS := fmt.Sprintf("%s.svc.cluster.local.", data.Cluster().Status.NamespaceName)

			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			cm.Data["Corefile"] = fmt.Sprintf(`
%s {
    forward . /etc/resolv.conf
    errors
}
%s {
    forward . %s
    errors
}
. {
  forward . /etc/resolv.conf
  errors
  health
}
`, seedClusterNamespaceDNS, data.Cluster().Spec.ClusterNetwork.DNSDomain, dnsIP)

			return cm, nil
		}
	}
}

// PodDisruptionBudgetCreator returns a func to create/update the apiserver PodDisruptionBudget
func PodDisruptionBudgetCreator() reconciling.NamedPodDisruptionBudgetCreatorGetter {
	return func() (string, reconciling.PodDisruptionBudgetCreator) {
		return resources.DNSResolverPodDisruptionBudetName, func(pdb *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
			minAvailable := intstr.FromInt(1)
			pdb.Spec = policyv1beta1.PodDisruptionBudgetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: resources.BaseAppLabel(resources.DNSResolverDeploymentName, nil),
				},
				MinAvailable: &minAvailable,
			}

			return pdb, nil
		}
	}
}
