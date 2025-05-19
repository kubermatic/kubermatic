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

package backgroundcontrollerresources

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	deploymentName = "kyverno-background-controller"
	kyvernoVersion = "v1.14.1"
)

// DeploymentReconciler returns the function to create and update the Kyverno background controller deployment.
func DeploymentReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return deploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Labels = map[string]string{
				"app.kubernetes.io/component": "background-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   kyvernoVersion,
			}

			// Deployment spec
			dep.Spec.Replicas = int32Ptr(1)
			dep.Spec.RevisionHistoryLimit = int32Ptr(10)
			dep.Spec.Strategy = appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
					MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "40%"},
				},
			}

			// Selector must match template labels
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component": "background-controller",
					"app.kubernetes.io/instance":  "kyverno",
					"app.kubernetes.io/part-of":   "kyverno",
				},
			}

			// Pod template
			dep.Spec.Template.ObjectMeta.Labels = map[string]string{
				"app.kubernetes.io/component": "background-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   kyvernoVersion,
			}

			// Pod spec
			dep.Spec.Template.Spec.DNSPolicy = corev1.DNSClusterFirst
			dep.Spec.Template.Spec.ServiceAccountName = backgroundControllerServiceAccountName

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
											Values:   []string{"background-controller"},
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
					Name: "uc-admin-kubeconfig",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
							// SecretName: resources.AdminKubeconfigSecretName,
							// SecretName: "kyverno-uc-sa-kubeconfig",
						},
					},
				},
			}

			// Main container
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            "controller",
					Image:           "reg.kyverno.io/kyverno/background-controller:" + kyvernoVersion,
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
						"--disableMetrics=false",
						"--otelConfig=prometheus",
						"--metricsPort=8000",
						"--resyncPeriod=15m",
						"--enableConfigMapCaching=true",
						"--enableDeferredLoading=true",
						"--maxAPICallResponseLength=2000000",
						"--loggingFormat=text",
						"--v=2",
						"--omitEvents=PolicyApplied,PolicySkipped",
						"--enablePolicyException=false",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "KYVERNO_SERVICEACCOUNT_NAME",
							Value: backgroundControllerServiceAccountName,
						},
						{
							Name:  "KYVERNO_DEPLOYMENT",
							Value: deploymentName,
						},
						{
							Name:  "INIT_CONFIG",
							Value: "kyverno",
						},
						{
							Name:  "METRICS_CONFIG",
							Value: "kyverno-metrics",
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
						AllowPrivilegeEscalation: boolPtr(false),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
						Privileged:             boolPtr(false),
						ReadOnlyRootFilesystem: boolPtr(true),
						RunAsNonRoot:           boolPtr(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
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

// Helper functions
func int32Ptr(i int32) *int32 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
