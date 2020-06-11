package etcd

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type cronJobCreatorData interface {
	Cluster() *kubermaticv1.Cluster
	ImageRegistry(string) string
	GetClusterRef() metav1.OwnerReference
}

// CronJobCreator returns the func to create/update the etcd defragger cronjob
func CronJobCreator(data cronJobCreatorData) reconciling.NamedCronJobCreatorGetter {
	return func() (string, reconciling.CronJobCreator) {
		return resources.EtcdDefragCronJobName, func(job *batchv1beta1.CronJob) (*batchv1beta1.CronJob, error) {
			command, err := defraggerCommand(data)
			if err != nil {
				return nil, err
			}

			job.Name = resources.EtcdDefragCronJobName
			job.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
			job.Spec.ConcurrencyPolicy = batchv1beta1.ForbidConcurrent
			var historyLimit int32
			job.Spec.SuccessfulJobsHistoryLimit = &historyLimit
			job.Spec.Schedule = "@every 3h"
			job.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
			job.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}
			job.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "defragger",
					Image:   data.ImageRegistry(resources.RegistryGCR) + "/etcd-development/etcd:" + ImageTag(data.Cluster()),
					Command: command,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.ApiserverEtcdClientCertificateSecretName,
							MountPath: "/etc/etcd/pki/client",
							ReadOnly:  true,
						},
					},
				},
			}

			job.Spec.JobTemplate.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: resources.ApiserverEtcdClientCertificateSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.ApiserverEtcdClientCertificateSecretName,
						},
					},
				},
			}

			return job, nil
		}
	}
}

type defraggerCommandTplData struct {
	ServiceName string
	Namespace   string
	CACertFile  string
	CertFile    string
	KeyFile     string
}

func defraggerCommand(data cronJobCreatorData) ([]string, error) {
	tpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(defraggerCommandTpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse etcd command template: %v", err)
	}

	tplData := defraggerCommandTplData{
		ServiceName: resources.EtcdServiceName,
		Namespace:   data.Cluster().Status.NamespaceName,
		CACertFile:  resources.CACertSecretKey,
		CertFile:    resources.ApiserverEtcdClientCertificateCertSecretKey,
		KeyFile:     resources.ApiserverEtcdClientCertificateKeySecretKey,
	}

	buf := bytes.Buffer{}
	if err := tpl.Execute(&buf, tplData); err != nil {
		return nil, err
	}

	return []string{
		"/bin/sh",
		"-ec",
		buf.String(),
	}, nil
}

const (
	defraggerCommandTpl = `etcdctl() {
ETCDCTL_API=3 /usr/local/bin/etcdctl \
  --command-timeout=60s \
  --endpoints https://$1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local.:2379 \
  --cacert /etc/etcd/pki/client/{{ .CACertFile }} \
  --cert /etc/etcd/pki/client/{{ .CertFile }} \
  --key /etc/etcd/pki/client/{{ .KeyFile }} \
  $2
}

for node in etcd-0 etcd-1 etcd-2; do
  etcdctl $node "endpoint health"

  if [ $? -eq 0 ]; then
    echo "Defragmenting $node..."
    etcdctl $node defrag
    sleep 30
  else
    echo "$node is not healthy, skipping defrag."
  fi
done`
)
