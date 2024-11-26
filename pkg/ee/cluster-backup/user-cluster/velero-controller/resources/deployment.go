//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2023 Kubermatic GmbH

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

package resources

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeploymentName = "velero"
)

type templateData interface {
	Cluster() *kubermaticv1.Cluster
	RewriteImage(image string) (string, error)
}

// DeploymentReconciler creates the velero deployment in the user cluster namespace.
func DeploymentReconciler(data templateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return DeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(DeploymentName, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}
			dep.Spec.Replicas = resources.Int32(1)

			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			volumes := getVolumes()
			volumeMounts := getVolumeMounts()

			kubernetes.EnsureLabels(&dep.Spec.Template, baseLabels)
			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				"prometheus.io/path":                                    "/metrics",
				"prometheus.io/port":                                    "8085",
				"prometheus.io/scrape":                                  "true",
				resources.ClusterLastRestartAnnotation:                  data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: "plugins,scratch",
			})

			dep.Spec.Template.Spec.Volumes = volumes
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{
				{
					Name:  "velero-velero-plugin-for-aws",
					Image: registry.Must(data.RewriteImage(fmt.Sprintf("velero/velero-plugin-for-aws:%s", pluginVersion))),
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "plugins",
							MountPath: "/target",
						},
					},
					ImagePullPolicy: corev1.PullPolicy("IfNotPresent"),
				},
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "velero",
					Image:   registry.Must(data.RewriteImage(fmt.Sprintf("velero/velero:%s", version))),
					Command: []string{"/velero"},
					Args: []string{
						"server",
						"--features=", // generated by running `velero install -o yaml --dry-run`
						"--uploader-type=kopia",
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8085,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "VELERO_SCRATCH_DIR",
							Value: "/scratch",
						},
						{
							Name:  "LD_LIBRARY_PATH",
							Value: "/plugins",
						},
						{
							Name:  "AWS_SHARED_CREDENTIALS_FILE",
							Value: fmt.Sprintf("/credentials/%s", defaultCloudCredentialsSecretKeyName),
						},
					},
					VolumeMounts:    volumeMounts,
					ImagePullPolicy: corev1.PullPolicy("IfNotPresent"),
				},
			}
			dep.Spec.Template.Spec.ServiceAccountName = resources.ClusterBackupServiceAccountName
			err := resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, nil, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
	}
}

func getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "plugins",
			MountPath: "/plugins",
		},
		{
			Name:      "scratch",
			MountPath: "/scratch",
		},
		{
			Name:      CloudCredentialsSecretName,
			MountPath: "/credentials",
		},
	}
}

func getVolumes() []corev1.Volume {
	vs := []corev1.Volume{
		{
			Name:         "plugins",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
		{
			Name:         "scratch",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
		{
			Name:         CloudCredentialsSecretName,
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: CloudCredentialsSecretName}},
		},
	}
	return vs
}
