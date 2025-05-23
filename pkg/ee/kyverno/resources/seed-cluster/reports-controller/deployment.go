//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2025 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package reportscontrollerresources

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	commonseedresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/common"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

// DeploymentReconciler returns the function to create and update the Kyverno reports controller deployment.
func DeploymentReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return commonseedresources.KyvernoReportsControllerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Labels = commonseedresources.KyvernoLabels(commonseedresources.ReportsControllerComponentNameLabel)

			// Deployment spec
			dep.Spec.Replicas = resources.Int32(2)
			dep.Spec.RevisionHistoryLimit = resources.Int32(10)
			dep.Spec.Strategy = appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
					MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "40%"},
				},
			}

			// Selector must match template labels
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: commonseedresources.KyvernoSelectorLabels(commonseedresources.ReportsControllerComponentNameLabel),
			}

			// Pod template
			dep.Spec.Template.Labels = commonseedresources.KyvernoLabels(commonseedresources.ReportsControllerComponentNameLabel)

			// Pod spec
			dep.Spec.Template.Spec.DNSPolicy = corev1.DNSClusterFirst
			dep.Spec.Template.Spec.ServiceAccountName = commonseedresources.KyvernoReportsControllerServiceAccountName

			// Pod anti-affinity
			dep.Spec.Template.Spec.Affinity = &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							Weight: 1,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{
										{
											Key:      "app.kubernetes.io/component",
											Operator: metav1.LabelSelectorOpIn,
											Values:   []string{commonseedresources.ReportsControllerComponentNameLabel},
										},
									},
								},
								TopologyKey: "kubernetes.io/hostname",
							},
						},
					},
				},
			}

			// Set volumes
			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "sigstore",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "uc-admin-kubeconfig",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
						},
					},
				},
			}

			// Main container
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            "controller",
					Image:           "reg.kyverno.io/kyverno/reports-controller:" + commonseedresources.KyvernoVersion,
					ImagePullPolicy: corev1.PullPolicy("IfNotPresent"),
					Ports: []corev1.ContainerPort{
						{
							Name:          "https",
							ContainerPort: 9443,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "metrics",
							ContainerPort: 8000,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Args: []string{
						"--kubeconfig=/etc/kubernetes/uc-admin-kubeconfig/kubeconfig",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "KYVERNO_SERVICEACCOUNT_NAME",
							Value: commonseedresources.KyvernoReportsControllerServiceAccountName,
						},
						{
							Name:  "KYVERNO_DEPLOYMENT",
							Value: commonseedresources.KyvernoReportsControllerDeploymentName,
						},
						{
							Name:  "INIT_CONFIG",
							Value: commonseedresources.KyvernoConfigMapName,
						},
						{
							Name:  "METRICS_CONFIG",
							Value: commonseedresources.KyvernoMetricsConfigMapName,
						},
						{
							Name: "KYVERNO_POD_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
						{
							Name: "KYVERNO_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
						{
							Name:  "TUF_ROOT",
							Value: "/.sigstore",
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: ptr.To(false),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
						Privileged:             ptr.To(false),
						ReadOnlyRootFilesystem: ptr.To(true),
						RunAsNonRoot:           ptr.To(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "sigstore",
							MountPath: "/.sigstore",
						},
						{
							Name:      "uc-admin-kubeconfig",
							MountPath: "/etc/kubernetes/uc-admin-kubeconfig",
							ReadOnly:  true,
						},
					},
				},
			}

			return dep, nil
		}
	}
}
