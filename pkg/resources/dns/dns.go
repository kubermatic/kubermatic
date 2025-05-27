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

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
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

// source: https://github.com/kubernetes/kubernetes/blob/vX.YY.0/cmd/kubeadm/app/constants/constants.go
func CoreDNSVersion(clusterVersion *semverlib.Version) string {
	switch fmt.Sprintf("%d.%d", clusterVersion.Major(), clusterVersion.Minor()) {
	case "1.27":
		fallthrough
	default:
		return "v1.10.1"
	}
}

func CoreDNSImage(clusterVersion *semverlib.Version) string {
	return fmt.Sprintf("coredns/coredns:%s", CoreDNSVersion(clusterVersion))
}

// ServiceReconciler returns the function to reconcile the DNS service.
func ServiceReconciler() reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
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

type deploymentReconcilerData interface {
	Cluster() *kubermaticv1.Cluster
	RewriteImage(string) (string, error)
	IsKonnectivityEnabled() bool
}

// DeploymentReconciler returns the function to create and update the DNS resolver deployment.
func DeploymentReconciler(data deploymentReconcilerData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.DNSResolverDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(resources.DNSResolverDeploymentName, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(2)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				"prometheus.io/path":                   "/metrics",
				"prometheus.io/scrape":                 "true",
				"prometheus.io/port":                   "9253",
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
			})

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name: resources.DNSResolverDeploymentName,
					// like etcd, this component follows the apiserver version and not the controller-manager version
					Image: registry.Must(data.RewriteImage(resources.RegistryK8S + "/" + CoreDNSImage(data.Cluster().Status.Versions.Apiserver.Semver()))),
					Args:  []string{"-conf", "/etc/coredns/Corefile"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.DNSResolverConfigMapName,
							MountPath: "/etc/coredns",
							ReadOnly:  true,
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
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
			}
			if !data.IsKonnectivityEnabled() {
				openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
				if err != nil {
					return nil, fmt.Errorf("failed to get openvpn sidecar for dns resolver: %w", err)
				}
				dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers,
					*openvpnSidecar,
				)
				defResourceRequirements[openvpnSidecar.Name] = openvpnSidecar.Resources.DeepCopy()
			}

			err := resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template.Spec.Volumes = getVolumes(data.IsKonnectivityEnabled())

			dep.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(resources.DNSResolverDeploymentName, kubermaticv1.AntiAffinityTypePreferred)

			return dep, nil
		}
	}
}

func getVolumes(isKonnectivityEnabled bool) []corev1.Volume {
	vs := []corev1.Volume{
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
	if !isKonnectivityEnabled {
		vs = append(vs, corev1.Volume{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNClientCertificatesSecretName,
				},
			},
		})
	}
	return vs
}

type configMapReconcilerData interface {
	Cluster() *kubermaticv1.Cluster
}

// ConfigMapReconciler returns a ConfigMap containing the cloud-config for the supplied data.
func ConfigMapReconciler(data configMapReconcilerData) reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
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

// PodDisruptionBudgetReconciler returns a func to create/update the apiserver PodDisruptionBudget.
func PodDisruptionBudgetReconciler() reconciling.NamedPodDisruptionBudgetReconcilerFactory {
	return func() (string, reconciling.PodDisruptionBudgetReconciler) {
		return resources.DNSResolverPodDisruptionBudetName, func(pdb *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
			minAvailable := intstr.FromInt(1)
			pdb.Spec = policyv1.PodDisruptionBudgetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: resources.BaseAppLabels(resources.DNSResolverDeploymentName, nil),
				},
				MinAvailable: &minAvailable,
			}

			return pdb, nil
		}
	}
}
