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

package metering

import (
	"fmt"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/controller/operator/common"
	"k8c.io/kubermatic/v3/pkg/ee/metering/prometheus"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

// CronJobReconciler returns the func to create/update the metering report cronjob.
func CronJobReconciler(reportName string, mrc *kubermaticv1.MeteringReportConfiguration, caBundleName string, getRegistry registry.ImageRewriter, namespace string) reconciling.NamedCronJobReconcilerFactory {
	return func() (string, reconciling.CronJobReconciler) {
		return reportName, func(job *batchv1.CronJob) (*batchv1.CronJob, error) {
			var args []string
			args = append(args, fmt.Sprintf("--ca-bundle=%s", "/opt/ca-bundle/ca-bundle.pem"))
			args = append(args, fmt.Sprintf("--prometheus-api=http://%s.%s.svc", prometheus.Name, namespace))
			args = append(args, fmt.Sprintf("--last-number-of-days=%d", mrc.Interval))
			args = append(args, fmt.Sprintf("--output-dir=%s", reportName))
			args = append(args, mrc.Types...)

			if job.Labels == nil {
				job.Labels = make(map[string]string)
			}
			job.Labels[common.NameLabel] = reportName
			job.Labels[common.ComponentLabel] = meteringName

			job.Spec.Schedule = mrc.Schedule
			job.Spec.JobTemplate.Spec.Parallelism = pointer.Int32(1)
			job.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = ""
			job.Spec.JobTemplate.Spec.Template.Spec.DeprecatedServiceAccount = ""
			job.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
			job.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			job.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            reportName,
					Image:           getMeteringImage(getRegistry),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/metering"},
					Args:            args,
					Env: []corev1.EnvVar{
						{
							Name: "S3_ENDPOINT",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: SecretName,
									},
									Key: Endpoint,
								},
							},
						},
						{
							Name: "S3_BUCKET",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: SecretName,
									},
									Key: Bucket,
								},
							},
						},
						{
							Name: "ACCESS_KEY_ID",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: SecretName,
									},
									Key: AccessKey,
								},
							},
						},
						{
							Name: "SECRET_ACCESS_KEY",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: SecretName,
									},
									Key: SecretKey,
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "ca-bundle",
							MountPath: "/opt/ca-bundle/",
							ReadOnly:  true,
						},
					},
				},
			}

			job.Spec.JobTemplate.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "ca-bundle",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: caBundleName,
							},
						},
					},
				},
			}
			return job, nil
		}
	}
}
