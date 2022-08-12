package cloudcontroller

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"
	appsv1 "k8s.io/api/apps/v1"
	_ "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

const (
	DigitalOceanCCMDeploymentName = "digital-ocean-cloud-controller-manager"
	DigitalOceanVersion           = "0.1.37"
)

func digitalOceanDeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return DigitalOceanCCMDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = DigitalOceanCCMDeploymentName
			dep.Labels = resources.BaseAppLabels(DigitalOceanCCMDeploymentName, nil)
			dep.Spec.Replicas = resources.Int32(1)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(DigitalOceanCCMDeploymentName, nil),
			}

			podLabels, err := data.GetPodTemplateLabels(DigitalOceanCCMDeploymentName, dep.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}

			dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err =
				resources.UserClusterDNSPolicyAndConfig(data)
			if err != nil {
				return nil, err
			}

			dep.Spec.Template.Spec.AutomountServiceAccountToken = pointer.Bool(false)

			dep.Spec.Template.Spec.Volumes = append(getVolumes(data.IsKonnectivityEnabled()),
				corev1.Volume{
					Name: resources.CloudConfigConfigMapName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: resources.CloudConfigConfigMapName,
							},
						},
					},
				})

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    ccmContainerName,
					Image:   data.ImageRegistry(resources.RegistryMCR) + "/oss/kubernetes/digital-ocean-cloud-controller-manager:v" + DigitalOceanVersion,
					Command: []string{"cloud-controller-manager"},
					Args:    getDigitalOceanFlags(data),
					VolumeMounts: append(getVolumeMounts(),
						corev1.VolumeMount{
							Name:      resources.CloudConfigConfigMapName,
							MountPath: "/etc/kubernetes/cloud",
							ReadOnly:  true,
						},
					),
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Scheme: corev1.URISchemeHTTP,
								Path:   "/healthz",
								Port:   intstr.FromInt(10267),
							},
						},
						SuccessThreshold:    1,
						FailureThreshold:    3,
						InitialDelaySeconds: 20,
						PeriodSeconds:       10,
						TimeoutSeconds:      5,
					},
				},
			}

			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				ccmContainerName: azureResourceRequirements.DeepCopy(),
			}

			if !data.IsKonnectivityEnabled() {
				openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, openvpnClientContainerName)
				if err != nil {
					return nil, fmt.Errorf("failed to get openvpn sidecar: %w", err)
				}
				dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, *openvpnSidecar)
				defResourceRequirements[openvpnSidecar.Name] = openvpnSidecar.Resources.DeepCopy()
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}
			return dep, nil
		}
	}
}

func getDigitalOceanFlags(data *resources.TemplateData) []string {
	flags := []string{
		"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
		"--leader-elect=true",
		"--v=5",
		"--cloud-config=/etc/kubernetes/cloud/config",
		"--cloud-provider=digitalocean",
	}

	if data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureCCMClusterName] {
		flags = append(flags, "--cluster-name", data.Cluster().Name)
	}
	return flags
}
