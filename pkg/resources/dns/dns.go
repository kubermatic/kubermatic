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

package dns

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"

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
			se.Spec.Selector = resources.BaseAppLabels(resources.DNSResolverDeploymentName, nil)
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
			dep.Labels = resources.BaseAppLabels(resources.DNSResolverDeploymentName, nil)
			dep.Spec.Replicas = resources.Int32(2)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(resources.DNSResolverDeploymentName, nil),
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			volumes := getVolumes()
			podLabels, err := data.GetPodTemplateLabels(resources.DNSResolverDeploymentName, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to get pod labels: %v", err)
			}

			dep.Spec.Template.ObjectMeta.Labels = podLabels

			if dep.Spec.Template.ObjectMeta.Annotations == nil {
				dep.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}

			dep.Spec.Template.ObjectMeta.Annotations["prometheus.io/scrape"] = "true"
			dep.Spec.Template.ObjectMeta.Annotations["prometheus.io/path"] = "/metrics"
			dep.Spec.Template.ObjectMeta.Annotations["prometheus.io/port"] = "9253"

			openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn sidecar for dns resolver: %v", err)
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				*openvpnSidecar,
				{
					Name:  resources.DNSResolverDeploymentName,
					Image: data.ImageRegistry(resources.RegistryK8SGCR) + "/coredns:1.3.1",
					Args:  []string{"-conf", "/etc/coredns/Corefile"},
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
			defResourceRequirements := map[string]*corev1.ResourceRequirements{
				resources.DNSResolverDeploymentName: defaultResourceRequirements.DeepCopy(),
				openvpnSidecar.Name:                 openvpnSidecar.Resources.DeepCopy(),
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
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
				},
			},
		},
		{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNClientCertificatesSecretName,
				},
			},
		},
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CASecretName,
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
  prometheus 0.0.0.0:9253
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
					MatchLabels: resources.BaseAppLabels(resources.DNSResolverDeploymentName, nil),
				},
				MinAvailable: &minAvailable,
			}

			return pdb, nil
		}
	}
}
