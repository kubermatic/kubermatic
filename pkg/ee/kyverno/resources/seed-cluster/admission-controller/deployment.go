//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0")
                     Copyright © 2021 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package resources

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	name = "kyverno-admission-controller"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("128Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("384Mi"),
		},
	}
)

// DeploymentReconciler returns the function to create and update the Kyverno admission controller deployment.
func DeploymentReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return name, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Labels = resources.BaseAppLabels(name, nil)
			dep.Spec.Replicas = resources.Int32(2)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(name, nil),
			}
			dep.Spec.Template.Labels = resources.BaseAppLabels(name, nil)
			dep.Spec.Template.Spec.ServiceAccountName = name

			volumes := []corev1.Volume{
				{
					Name: "sigstore",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}

			volumeMounts := []corev1.VolumeMount{
				{
					Name:      "sigstore",
					MountPath: "/.sigstore",
				},
			}

			dep.Spec.Template.Spec.Volumes = volumes

			args := []string{
				"--caSecretName=kyverno-svc.kyverno.svc.kyverno-tls-ca",
				"--tlsSecretName=kyverno-svc.kyverno.svc.kyverno-tls-pair",
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
				"--dumpPayload=false",
				"--forceFailurePolicyIgnore=false",
				"--generateValidatingAdmissionPolicy=false",
				"--dumpPatches=false",
				"--maxAPICallResponseLength=2000000",
				"--loggingFormat=text",
				"--v=2",
				"--omitEvents=PolicyApplied,PolicySkipped",
				"--enablePolicyException=false",
				"--protectManagedResources=false",
				"--allowInsecureRegistry=false",
				"--registryCredentialHelpers=default,google,amazon,azure,github",
				"--enableReporting=validate,mutate,mutateExisting,imageVerify,generate",
				// kubeconfig WIP
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:         name,
					Image:        "ghcr.io/kyverno/kyverno:v1.13.2",
					Args:         args,
					VolumeMounts: volumeMounts,
					Resources:    defaultResourceRequirements,
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health/liveness",
								Port:   intstr.FromInt(9443),
								Scheme: corev1.URISchemeHTTPS,
							},
						},
						InitialDelaySeconds: 15,
						PeriodSeconds:       30,
						FailureThreshold:    2,
						TimeoutSeconds:      5,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health/readiness",
								Port:   intstr.FromInt(9443),
								Scheme: corev1.URISchemeHTTPS,
							},
						},
						InitialDelaySeconds: 5,
						PeriodSeconds:       10,
						FailureThreshold:    6,
						TimeoutSeconds:      5,
					},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 9443,
							Name:          "https",
							Protocol:      corev1.ProtocolTCP,
						},
						{
							ContainerPort: 8000,
							Name:          "metrics-port",
							Protocol:      corev1.ProtocolTCP,
						},
					},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: resources.Bool(false),
						ReadOnlyRootFilesystem:   resources.Bool(true),
						RunAsNonRoot:             resources.Bool(true),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
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
							Value: name,
						},
						{
							Name:  "KYVERNO_DEPLOYMENT",
							Value: name,
						},
						{
							Name:  "TUF_ROOT",
							Value: "/.sigstore",
						},
					},
				},
			}

			return dep, nil
		}
	}
}
