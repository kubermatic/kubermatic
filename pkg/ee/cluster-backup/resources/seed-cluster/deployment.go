//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

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

package seedclusterresources

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterBackupName                 = "velero-cluster-backup"
	clusterbackupKubeConfigSecretName = "velero-kubeconfig"
	cloudCredentialsSecretName        = "velero-cloud-credentials"

	veleroImage = "velero/velero:v1.12.0"
)

// DeploymentReconciler creates the velero deployment in the user cluster namespace.
func DeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return clusterBackupName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Labels = resources.BaseAppLabels(clusterBackupName, nil)

			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(clusterBackupName, nil),
			}
			dep.Spec.Replicas = resources.Int32(1)

			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			volumes := getVolumes()
			volumeMounts := getVolumeMounts()

			podLabels, err := data.GetPodTemplateLabels(clusterBackupName, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %w", err)
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					"prometheus.io/path":   "/metrics",
					"prometheus.io/port":   "8085",
					"prometheus.io/scrape": "true",
				},
			}

			dep.Spec.Template.Spec.Volumes = volumes
			dep.Spec.Template.Spec.InitContainers = []corev1.Container{
				{
					Name:  "velero-velero-plugin-for-aws",
					Image: "velero/velero-plugin-for-aws:v1.0.0",
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
					Image:   veleroImage,
					Command: []string{"/velero"},
					Args: []string{
						"server",
						"--features=",
						"--uploader-type=kopia",
						fmt.Sprintf("--namespace=%s", resources.ClusterBackupNamespaceName),
						"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8085,
							Protocol:      corev1.ProtocolSCTP,
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
							Name:  "GOOGLE_APPLICATION_CREDENTIALS",
							Value: "/credentials/cloud",
						},
						{
							Name:  "AWS_SHARED_CREDENTIALS_FILE",
							Value: "/credentials/cloud",
						},
						{
							Name:  "AZURE_CREDENTIALS_FILE",
							Value: "/credentials/cloud",
						},
						{
							Name:  "ALIBABA_CLOUD_CREDENTIALS_FILE",
							Value: "/credentials/cloud",
						},
						// looks like velero has a bug where it doesn't use the provided kubeconfig for some operations
						// and falls back to in-cluster credentials. This is a workaround.
						{
							Name:  "KUBECONFIG",
							Value: "/etc/kubernetes/kubeconfig/kubeconfig",
						},
					},
					VolumeMounts:    volumeMounts,
					ImagePullPolicy: corev1.PullPolicy("IfNotPresent"),
				},
			}
			dep.Spec.Template.Spec.ServiceAccountName = "default"
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, nil, nil, dep.Annotations)
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
			Name:      resources.ClusterbackupKubeconfigSecretName,
			MountPath: "/etc/kubernetes/kubeconfig",
			ReadOnly:  true,
		},
		{
			Name:      resources.CASecretName,
			MountPath: "/etc/kubernetes/pki/ca",
			ReadOnly:  true,
		},
		{
			Name:      resources.CABundleConfigMapName,
			MountPath: "/etc/kubernetes/pki/ca-bundle",
			ReadOnly:  true,
		},
		{
			Name:      "plugins",
			MountPath: "/plugins",
		},
		{
			Name:      "scratch",
			MountPath: "/scratch",
		},
		{
			Name:      cloudCredentialsSecretName,
			MountPath: "/credentials",
		},
	}
}

func getVolumes() []corev1.Volume {
	vs := []corev1.Volume{
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
		{
			Name: resources.CABundleConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.CABundleConfigMapName,
					},
				},
			},
		},
		{
			Name: resources.ClusterbackupKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ClusterbackupKubeconfigSecretName,
				},
			},
		},

		{
			Name:         "plugins",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
		{
			Name:         "scratch",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
		{
			Name:         cloudCredentialsSecretName,
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: cloudCredentialsSecretName}},
		},
	}
	return vs
}
