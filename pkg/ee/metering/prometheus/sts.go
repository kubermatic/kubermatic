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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/registry"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func getPrometheusImage(overwriter registry.ImageRewriter) string {
	return registry.Must(overwriter(resources.RegistryQuay + "/prometheus/prometheus:v2.37.0"))
}

// prometheusStatefulSet creates a StatefulSet for prometheus.
func prometheusStatefulSet(getRegistry registry.ImageRewriter, seed *kubermaticv1.Seed) reconciling.NamedStatefulSetReconcilerFactory {
	return func() (string, reconciling.StatefulSetCreator) {
		return Name, func(sts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
			if sts.Labels == nil {
				sts.Labels = make(map[string]string)
			}
			sts.Labels[common.NameLabel] = Name

			sts.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{common.NameLabel: Name},
			}
			sts.Spec.Replicas = pointer.Int32(1)
			sts.Spec.RevisionHistoryLimit = pointer.Int32(10)
			sts.Spec.ServiceName = Name + "-headless"
			sts.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
			sts.Spec.PodManagementPolicy = appsv1.OrderedReadyPodManagement

			if sts.Spec.Template.Labels == nil {
				sts.Spec.Template.Labels = make(map[string]string)
			}

			sts.Spec.Template.Labels[common.NameLabel] = Name
			sts.Spec.Template.Name = Name
			sts.Spec.Template.Spec.ServiceAccountName = Name

			sts.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            Name,
					Image:           getPrometheusImage(getRegistry),
					ImagePullPolicy: "IfNotPresent",
					Args:            []string{"--storage.tsdb.retention.time=90d", "--config.file=/etc/config/prometheus.yml", "--storage.tsdb.path=/data", "--web.console.libraries=/etc/prometheus/console_libraries", "--web.console.templates=/etc/prometheus/consoles", "--web.enable-lifecycle"},
					Ports:           []corev1.ContainerPort{{ContainerPort: 9090, Protocol: corev1.ProtocolTCP}},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("250m"),
							corev1.ResourceMemory: resource.MustParse("512Mi")},
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

			sts.Spec.Template.Spec.DNSPolicy = "ClusterFirst"
			sts.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				FSGroup:      pointer.Int64(65534),
				RunAsGroup:   pointer.Int64(65534),
				RunAsNonRoot: pointer.Bool(false),
				RunAsUser:    pointer.Int64(65534),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			sts.Spec.Template.Spec.TerminationGracePeriodSeconds = pointer.Int64(300)

			sts.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "config-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							DefaultMode:          pointer.Int32(420),
							LocalObjectReference: corev1.LocalObjectReference{Name: Name},
						},
					},
				},
			}

			pvcStorageSize, err := resource.ParseQuantity(seed.Spec.Metering.StorageSize)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value of prometheus pvc storage size %q: %w", seed.Spec.Metering.StorageSize, err)
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
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: pvcStorageSize,
							},
						},
						VolumeMode:       &volumeMode,
						StorageClassName: pointer.String(seed.Spec.Metering.StorageClassName),
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
