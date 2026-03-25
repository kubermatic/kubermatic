//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2024 Kubermatic GmbH

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

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	DaemonSetName = "node-agent"
)

var (
	// The "name" label is required here: it's used by Velero to detect the daemonset pods on the nodes,
	// if it's not there Velero will partially fail to do the backup:
	// https://github.com/vmware-tanzu/velero/blob/b30a679e5b1c2cbd9021e1301580f2359ef981bf/pkg/nodeagent/node_agent.go#L84
	veleroAdditionalLabels = map[string]string{
		"app.kubernetes.io/name": DaemonSetName,
		"name":                   DaemonSetName,
	}

	// veleroAdditionalPodLabels contains labels that should only be on node-agent pods (not in the selector).
	// The "role" label is required since Velero v1.17 uses "role=node-agent" to find node-agent pods in isRunningInNode:
	// https://github.com/vmware-tanzu/velero/blob/v1.17.1/pkg/nodeagent/node_agent.go
	// This is separate from veleroAdditionalLabels because the DaemonSet selector is immutable.
	veleroAdditionalPodLabels = map[string]string{
		"role": DaemonSetName,
	}
)

// DaemonSetReconciler returns the function to create and update the Velero node-agent DaemonSet.
func DaemonSetReconciler(data templateData) reconciling.NamedDaemonSetReconcilerFactory {
	return func() (string, reconciling.DaemonSetReconciler) {
		return DaemonSetName, func(ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
			baseLabels := resources.BaseAppLabels(DaemonSetName, map[string]string{"component": "velero"})
			kubernetes.EnsureLabels(ds, baseLabels)

			podLabels := resources.BaseAppLabels(DaemonSetName, veleroAdditionalLabels)
			ds.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: podLabels,
			}

			kubernetes.EnsureLabels(&ds.Spec.Template, podLabels)
			kubernetes.EnsureLabels(&ds.Spec.Template, veleroAdditionalPodLabels)
			kubernetes.EnsureAnnotations(&ds.Spec.Template, map[string]string{
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: "scratch",
			})

			ds.Spec.Template.Spec = corev1.PodSpec{
				Containers: getContainers(data),
				DNSPolicy:  corev1.DNSClusterFirst,
				Volumes: []corev1.Volume{
					{
						Name: "host-pods",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/lib/kubelet/pods",
								Type: ptr.To[corev1.HostPathType](corev1.HostPathUnset),
							}},
					},
					{
						Name:         "scratch",
						VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
					},
					{
						Name:         CloudCredentialsSecretName,
						VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: CloudCredentialsSecretName}},
					},
				},
				RestartPolicy:                 corev1.RestartPolicyAlways,
				TerminationGracePeriodSeconds: ptr.To[int64](30),
				SecurityContext: &corev1.PodSecurityContext{
					RunAsUser: ptr.To[int64](0),
				},
				SchedulerName: corev1.DefaultSchedulerName,
			}

			ds.Spec.Template.Spec.ServiceAccountName = resources.ClusterBackupServiceAccountName
			ds.Spec.Template.Spec.DeprecatedServiceAccount = resources.ClusterBackupServiceAccountName

			return ds, nil
		}
	}
}

func getContainers(data templateData) []corev1.Container {
	return []corev1.Container{
		{
			Name:            DaemonSetName,
			Image:           registry.Must(data.RewriteImage(fmt.Sprintf("quay.io/kubermatic-mirror/images/velero:%s", version))),
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command: []string{
				"/velero",
			},
			Args: []string{
				"node-agent",
				"server",
				"--features=",
			},
			Env: []corev1.EnvVar{
				{
					Name: "NODE_NAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "spec.nodeName",
						},
					},
				},
				{
					Name:  "VELERO_NAMESPACE",
					Value: resources.ClusterBackupNamespaceName,
				},
				{
					Name:  "VELERO_SCRATCH_DIR",
					Value: "/scratch",
				},
				{
					Name:  "AWS_SHARED_CREDENTIALS_FILE",
					Value: fmt.Sprintf("/credentials/%s", defaultCloudCredentialsSecretKeyName),
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:             "host-pods",
					MountPath:        "/host_pods",
					MountPropagation: ptr.To[corev1.MountPropagationMode](corev1.MountPropagationHostToContainer),
				},
				{
					Name:      "scratch",
					MountPath: "/scratch",
				},
				{
					Name:      CloudCredentialsSecretName,
					MountPath: "/credentials",
				},
			},
			// The node-agent purposefully does not have resource constraints since Velero 1.14, see
			// https://github.com/vmware-tanzu/velero/issues/7391
			// Resources: corev1.ResourceRequirements{},
		},
	}
}
