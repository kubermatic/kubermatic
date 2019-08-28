package etcd

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
			corev1.ResourceCPU:    resource.MustParse("50m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2Gi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}
)

const (
	name    = "etcd"
	dataDir = "/var/run/etcd/pod_${POD_NAME}/"
	// ImageTag defines the image tag to use for the etcd image
	ImageTag = "v3.3.15"
)

type etcdStatefulSetCreatorData interface {
	Cluster() *kubermaticv1.Cluster
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	HasEtcdOperatorService() (bool, error)
	ImageRegistry(string) string
	EtcdDiskSize() resource.Quantity
	GetClusterRef() metav1.OwnerReference
	SupportsFailureDomainZoneAntiAffinity() bool
}

// StatefulSetCreator returns the function to reconcile the etcd StatefulSet
func StatefulSetCreator(data etcdStatefulSetCreatorData, enableDataCorruptionChecks bool) reconciling.NamedStatefulSetCreatorGetter {
	return func() (string, reconciling.StatefulSetCreator) {
		return resources.EtcdStatefulSetName, func(set *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
			set.Name = resources.EtcdStatefulSetName

			set.Spec.Replicas = resources.Int32(resources.EtcdClusterSize)
			set.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
			set.Spec.PodManagementPolicy = appsv1.ParallelPodManagement
			set.Spec.ServiceName = resources.EtcdServiceName
			set.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			baseLabels := getBasePodLabels(data.Cluster())
			set.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			volumes := getVolumes()
			podLabels, err := data.GetPodTemplateLabels(resources.EtcdStatefulSetName, volumes, baseLabels)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}

			set.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Name:   name,
				Labels: podLabels,
			}

			// For migration purpose.
			// We switched from the etcd-operator to a simple etcd-StatefulSet. Therefore we need to migrate the data.
			migrate, err := data.HasEtcdOperatorService()
			if err != nil {
				return nil, fmt.Errorf("failed to check if we need to include the etcd-operator migration code: %v", err)
			}

			etcdStartCmd, err := getEtcdCommand(data.Cluster().Name, data.Cluster().Status.NamespaceName, migrate, enableDataCorruptionChecks)
			if err != nil {
				return nil, err
			}
			resourceRequirements := defaultResourceRequirements
			if data.Cluster().Spec.ComponentsOverride.Etcd.Resources != nil {
				resourceRequirements = *data.Cluster().Spec.ComponentsOverride.Etcd.Resources
			}
			set.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    name,
					Image:   data.ImageRegistry(resources.RegistryGCR) + "/etcd-development/etcd:" + ImageTag,
					Command: etcdStartCmd,
					Env: []corev1.EnvVar{
						{
							Name: "POD_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									APIVersion: "v1",
									FieldPath:  "metadata.name",
								},
							},
						},
						{
							Name: "POD_IP",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									APIVersion: "v1",
									FieldPath:  "status.podIP",
								},
							},
						},
						{
							Name:  "ETCDCTL_API",
							Value: "3",
						},
					},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 2379,
							Protocol:      corev1.ProtocolTCP,
							Name:          "client",
						},
						{
							ContainerPort: 2380,
							Protocol:      corev1.ProtocolTCP,
							Name:          "peer",
						},
					},
					Resources: resourceRequirements,
					ReadinessProbe: &corev1.Probe{
						TimeoutSeconds:      10,
						PeriodSeconds:       30,
						SuccessThreshold:    1,
						FailureThreshold:    3,
						InitialDelaySeconds: 15,
						Handler: corev1.Handler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"/usr/local/bin/etcdctl",
									"--command-timeout", "10s",
									"--cacert", "/etc/etcd/pki/ca/ca.crt",
									"--cert", "/etc/etcd/pki/client/apiserver-etcd-client.crt",
									"--key", "/etc/etcd/pki/client/apiserver-etcd-client.key",
									"--endpoints", "https://127.0.0.1:2379", "endpoint", "health",
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "data",
							MountPath: "/var/run/etcd",
						},
						{
							Name:      resources.EtcdTLSCertificateSecretName,
							MountPath: "/etc/etcd/pki/tls",
						},
						{
							Name:      resources.CASecretName,
							MountPath: "/etc/etcd/pki/ca",
						},
						{
							Name:      resources.ApiserverEtcdClientCertificateSecretName,
							MountPath: "/etc/etcd/pki/client",
							ReadOnly:  true,
						},
					},
				},
			}

			set.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(resources.EtcdStatefulSetName, data.Cluster().Name)
			if data.SupportsFailureDomainZoneAntiAffinity() {
				antiAffinities := set.Spec.Template.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
				antiAffinities = append(antiAffinities, resources.FailureDomainZoneAntiAffinity(resources.EtcdStatefulSetName, data.Cluster().Name))
				set.Spec.Template.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = antiAffinities
			}

			set.Spec.Template.Spec.Volumes = volumes

			// Make sure, we don't change size of existing pvc's
			// Phase needs to be taken from an existing
			diskSize := data.EtcdDiskSize()
			if len(set.Spec.VolumeClaimTemplates) == 0 {
				set.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "data",
							OwnerReferences: []metav1.OwnerReference{data.GetClusterRef()},
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							StorageClassName: resources.String("kubermatic-fast"),
							AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{corev1.ResourceStorage: diskSize},
							},
						},
					},
				}
			}

			return set, nil
		}
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.EtcdTLSCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.EtcdTLSCertificateSecretName,
				},
			},
		},
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
			Name: resources.ApiserverEtcdClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.ApiserverEtcdClientCertificateSecretName,
				},
			},
		},
	}
}

func getBasePodLabels(cluster *kubermaticv1.Cluster) map[string]string {
	additionalLabels := map[string]string{
		"cluster": cluster.Name,
	}
	return resources.BaseAppLabel(resources.EtcdStatefulSetName, additionalLabels)
}

type commandTplData struct {
	ServiceName           string
	Namespace             string
	Token                 string
	DataDir               string
	Migrate               bool
	EnableCorruptionCheck bool
}

func getEtcdCommand(name, namespace string, migrate, enableCorruptionCheck bool) ([]string, error) {
	tpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(etcdStartCommandTpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse etcd command template: %v", err)
	}

	tplData := commandTplData{
		ServiceName:           resources.EtcdServiceName,
		Token:                 name,
		Namespace:             namespace,
		DataDir:               dataDir,
		Migrate:               migrate,
		EnableCorruptionCheck: enableCorruptionCheck,
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
	etcdStartCommandTpl = `export MASTER_ENDPOINT="https://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2379"

{{ if .Migrate }}
# If we're already initialized
if [ -d "{{ .DataDir }}" ]; then
    echo "we're already initialized"
    export INITIAL_STATE="existing"
    if [ "${POD_NAME}" = "etcd-0" ]; then
        export INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"
    fi
    if [ "${POD_NAME}" = "etcd-1" ]; then
        export INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-1=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"
    fi
    if [ "${POD_NAME}" = "etcd-2" ]; then
        export INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-1=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-2=http://etcd-2.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"
    fi
else
    if [ "${POD_NAME}" = "etcd-0" ]; then
        echo "i'm etcd-0. I do the restore"
        etcdctl --endpoints http://etcd-cluster-client:2379 snapshot save snapshot.db
        etcdctl snapshot restore snapshot.db \
            --name etcd-0 \
            --data-dir="{{ .DataDir }}" \
            --initial-cluster="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380" \
            --initial-cluster-token="{{ .Token }}" \
            --initial-advertise-peer-urls http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380
        echo "restored from snapshot"
        export INITIAL_STATE="new"
        export INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"
    fi

    export ETCD_CERT_ARGS="--cacert /etc/etcd/pki/ca/ca.crt --cert /etc/etcd/pki/client/apiserver-etcd-client.crt --key /etc/etcd/pki/client/apiserver-etcd-client.key"
    if [ "${POD_NAME}" = "etcd-1" ]; then
        echo "i'm etcd-1. I join as new member as soon as etcd-0 comes up"
        etcdctl ${ETCD_CERT_ARGS} --endpoints ${MASTER_ENDPOINT} member add etcd-1 --peer-urls=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380
        echo "added etcd-1 to members"
        export INITIAL_STATE="existing"
        export INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-1=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"
    fi

    if [ "${POD_NAME}" = "etcd-2" ]; then
        echo "i'm etcd-2. I join as new member as soon as we have 2 existing & healthy members"
        until etcdctl ${ETCD_CERT_ARGS} --endpoints ${MASTER_ENDPOINT} member list | grep -q etcd-1; do sleep 1; echo "Waiting for etcd-1"; done
        etcdctl ${ETCD_CERT_ARGS} --endpoints ${MASTER_ENDPOINT} member add etcd-2 --peer-urls=http://etcd-2.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380
        echo "added etcd-2 to members"
        export INITIAL_STATE="existing"
        export INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-1=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-2=http://etcd-2.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"
    fi
fi

{{ else }}
export INITIAL_STATE="new"
export INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-1=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-2=http://etcd-2.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"
{{ end }}

echo "initial-state: ${INITIAL_STATE}"
echo "initial-cluster: ${INITIAL_CLUSTER}"

exec /usr/local/bin/etcd \
    --name=${POD_NAME} \
    --data-dir="{{ .DataDir }}" \
    --initial-cluster=${INITIAL_CLUSTER} \
    --initial-cluster-token="{{ .Token }}" \
    --initial-cluster-state=${INITIAL_STATE} \
    --advertise-client-urls "https://${POD_NAME}.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2379,https://${POD_IP}:2379" \
    --listen-client-urls "https://${POD_IP}:2379,https://127.0.0.1:2379" \
    --listen-peer-urls "http://${POD_IP}:2380" \
    --initial-advertise-peer-urls "http://${POD_NAME}.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380" \
    --trusted-ca-file /etc/etcd/pki/ca/ca.crt \
    --client-cert-auth \
    --cert-file /etc/etcd/pki/tls/etcd-tls.crt \
    --key-file /etc/etcd/pki/tls/etcd-tls.key \
{{- if .EnableCorruptionCheck }}
    --experimental-initial-corrupt-check=true \
    --experimental-corrupt-check-time=10m \
{{- end }}
    --auto-compaction-retention=8
`
)
