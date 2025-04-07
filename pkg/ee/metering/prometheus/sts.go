//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

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

package prometheus

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	version = "v2.51.1"
)

func getPrometheusImage(overwriter registry.ImageRewriter) string {
	return registry.Must(overwriter(resources.RegistryQuay + "/prometheus/prometheus:" + version))
}

// prometheusStatefulSet creates a StatefulSet for prometheus.
func PrometheusStatefulSet(getRegistry registry.ImageRewriter, seed *kubermaticv1.Seed) reconciling.NamedStatefulSetReconcilerFactory {
	return func() (string, reconciling.StatefulSetReconciler) {
		return Name, func(sts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
			basicLabels := map[string]string{common.NameLabel: Name}
			kubernetes.EnsureLabels(sts, basicLabels)

			sts.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: basicLabels,
			}
			sts.Spec.Replicas = ptr.To[int32](1)
			sts.Spec.ServiceName = Name + "-headless"
			sts.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
			sts.Spec.PodManagementPolicy = appsv1.OrderedReadyPodManagement

			sts.Spec.Template.Name = Name
			kubernetes.EnsureLabels(&sts.Spec.Template, basicLabels)
			sts.Spec.Template.Spec.ServiceAccountName = Name

			sts.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            Name,
					Image:           getPrometheusImage(getRegistry),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args: []string{
						fmt.Sprintf("--storage.tsdb.retention.time=%dd", seed.Spec.Metering.RetentionDays),
						"--config.file=/etc/config/prometheus.yml",
						"--storage.tsdb.path=/data",
						"--web.enable-lifecycle",
					},
					Ports: []corev1.ContainerPort{{ContainerPort: 9090, Protocol: corev1.ProtocolTCP}},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("250m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/-/healthy",
								Port:   intstr.FromInt(9090),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 30,
						TimeoutSeconds:      10,
						PeriodSeconds:       15,
						SuccessThreshold:    1,
						FailureThreshold:    3,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/-/ready",
								Port:   intstr.FromInt(9090),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 30,
						TimeoutSeconds:      4,
						PeriodSeconds:       5,
						SuccessThreshold:    1,
						FailureThreshold:    3,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "config-volume",
							MountPath: "/etc/config",
						},
						{
							Name:      "storage",
							MountPath: "/data",
						},
					},
				},
			}

			sts.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				FSGroup:      ptr.To[int64](65534),
				RunAsGroup:   ptr.To[int64](65534),
				RunAsNonRoot: ptr.To(false),
				RunAsUser:    ptr.To[int64](65534),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			sts.Spec.Template.Spec.TerminationGracePeriodSeconds = ptr.To[int64](300)

			sts.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "config-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							DefaultMode:          ptr.To[int32](420),
							LocalObjectReference: corev1.LocalObjectReference{Name: Name},
						},
					},
				},
			}

			pvcStorageSize, err := resource.ParseQuantity(seed.Spec.Metering.StorageSize)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value of PVC storage size %q: %w", seed.Spec.Metering.StorageSize, err)
			}

			volumeMode := corev1.PersistentVolumeFilesystem

			sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "storage",
						Namespace: seed.Namespace,
						Labels:    map[string]string{common.NameLabel: Name},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: pvcStorageSize,
							},
						},
						VolumeMode:       &volumeMode,
						StorageClassName: ptr.To(seed.Spec.Metering.StorageClassName),
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimPending,
					},
				},
			}

			return sts, nil
		}
	}
}
