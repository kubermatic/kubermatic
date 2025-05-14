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

package admissioncontrollerresources

import (
	"fmt"

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
	deploymentName = "kyverno-admission-controller"
	kyvernoVersion = "v1.14.1"
)

// DeploymentReconciler returns the function to create and update the Kyverno admission controller deployment.
func DeploymentReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return deploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Labels = map[string]string{
				"app.kubernetes.io/component": "admission-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   kyvernoVersion,
			}

			// Deployment spec
			dep.Spec.Replicas = int32Ptr(2)
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
					"app.kubernetes.io/component": "admission-controller",
					"app.kubernetes.io/instance":  "kyverno",
					"app.kubernetes.io/part-of":   "kyverno",
				},
			}

			// Pod template
			dep.Spec.Template.ObjectMeta.Labels = map[string]string{
				"app.kubernetes.io/component": "admission-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   kyvernoVersion,
			}

			// Pod spec
			dep.Spec.Template.Spec.DNSPolicy = corev1.DNSClusterFirst
			dep.Spec.Template.Spec.ServiceAccountName = admissionControllerServiceAccountName

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
											Values:   []string{"admission-controller"},
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
			namespace := cluster.Status.NamespaceName
			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "uc-admin-kubeconfig",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
						},
					},
				},
				{
					Name: "sigstore",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}

			// Init container
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{
				{
					Name:            "kyverno-pre",
					Image:           "reg.kyverno.io/kyverno/kyvernopre:" + kyvernoVersion,
					ImagePullPolicy: corev1.PullPolicy("IfNotPresent"),
					Args: []string{
						// "--kubeconfig=/etc/kubernetes/uc-admin-kubeconfig/kubeconfig",
						"--loggingFormat=text",
						"--v=2",
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("10m"),
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
					Env: []corev1.EnvVar{
						{
							Name:  "KYVERNO_SERVICEACCOUNT_NAME",
							Value: admissionControllerServiceAccountName,
						},
						{
							Name:  "KYVERNO_ROLE_NAME",
							Value: admissionControllerRoleName,
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
							Name: "KYVERNO_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
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
							Name:  "KYVERNO_DEPLOYMENT",
							Value: deploymentName,
						},
						{
							Name:  "KYVERNO_SVC",
							Value: serviceName,
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

			// Main container
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            "kyverno",
					Image:           "reg.kyverno.io/kyverno/kyverno:" + kyvernoVersion,
					ImagePullPolicy: corev1.PullPolicy("IfNotPresent"),
					Args: []string{
						// "--leaderElect=false",
						"--kubeconfig=/etc/kubernetes/uc-admin-kubeconfig/kubeconfig",
						fmt.Sprintf("--caSecretName=kyverno-svc.%s.svc.kyverno-tls-ca", namespace),
						fmt.Sprintf("--tlsSecretName=kyverno-svc.%s.svc.kyverno-tls-pair", namespace),
						// "--backgroundServiceAccountName=system:serviceaccount:kyverno-system-uc1:kyverno-background-controller-uc1-sa",
						// "--reportsServiceAccountName=system:serviceaccount:kyverno-system-uc1:kyverno-reports-controller-uc1-sa",
						"--servicePort=443",
						"--webhookServerPort=9443",
						"--resyncPeriod=15m",
						"--disableMetrics=false",
						"--otelConfig=prometheus",
						"--metricsPort=8000",
						"--admissionReports=true",
						"--maxAdmissionReports=1000",
						"--autoUpdateWebhooks=true",
						"--enableConfigMapCaching=true",
						"--enableDeferredLoading=true",
						"--forceFailurePolicyIgnore=false",
						"--maxAPICallResponseLength=2000000",
						"--loggingFormat=text",
						"--v=4",
						"--omitEvents=PolicyApplied,PolicySkipped",
						"--enablePolicyException=false",
						"--protectManagedResources=false",
						"--enableReporting=validate,mutate,mutateExisting,imageVerify,generate",
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("384Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
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
					Ports: []corev1.ContainerPort{
						{
							Name:          "https",
							ContainerPort: 9443,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "metrics-port",
							ContainerPort: 8000,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "INIT_CONFIG",
							Value: "kyverno",
						},
						{
							Name:  "METRICS_CONFIG",
							Value: "kyverno-metrics",
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
							Name: "KYVERNO_POD_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
						{
							Name:  "KYVERNO_SERVICEACCOUNT_NAME",
							Value: admissionControllerServiceAccountName,
						},
						{
							Name:  "KYVERNO_ROLE_NAME",
							Value: admissionControllerRoleName,
						},
						{
							Name:  "KYVERNO_SVC",
							Value: serviceName,
						},
						{
							Name:  "TUF_ROOT",
							Value: "/.sigstore",
						},
						{
							Name:  "KYVERNO_DEPLOYMENT",
							Value: deploymentName,
						},
					},
					StartupProbe: &corev1.Probe{
						FailureThreshold: 20,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health/liveness",
								Port:   intstr.FromInt(9443),
								Scheme: corev1.URISchemeHTTPS,
							},
						},
						InitialDelaySeconds: 2,
						PeriodSeconds:       6,
					},
					LivenessProbe: &corev1.Probe{
						FailureThreshold: 2,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health/liveness",
								Port:   intstr.FromInt(9443),
								Scheme: corev1.URISchemeHTTPS,
							},
						},
						InitialDelaySeconds: 15,
						PeriodSeconds:       30,
						SuccessThreshold:    1,
						TimeoutSeconds:      5,
					},
					ReadinessProbe: &corev1.Probe{
						FailureThreshold: 6,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health/readiness",
								Port:   intstr.FromInt(9443),
								Scheme: corev1.URISchemeHTTPS,
							},
						},
						InitialDelaySeconds: 5,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      5,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "uc-admin-kubeconfig",
							MountPath: "/etc/kubernetes/uc-admin-kubeconfig",
							ReadOnly:  false,
						},
						{
							Name:      "sigstore",
							MountPath: "/.sigstore",
							ReadOnly:  false,
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
