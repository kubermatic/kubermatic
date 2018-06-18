package etcd

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	defaultEtcdMemoryRequest = resource.MustParse("256Mi")
	defaultEtcdCPURequest    = resource.MustParse("50m")
	defaultEtcdMemoryLimit   = resource.MustParse("1Gi")
	defaultEtcdCPULimit      = resource.MustParse("100m")
)

const (
	name    = "etcd"
	dataDir = "/var/run/etcd/pod_${POD_NAME}/"
)

// StatefulSet returns the etcd StatefulSet
func StatefulSet(data *resources.TemplateData, existing *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	var set *appsv1.StatefulSet
	if existing != nil {
		set = existing
	} else {
		set = &appsv1.StatefulSet{}
	}

	set.Name = resources.EtcdStatefulSetName
	set.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	set.Spec.Replicas = resources.Int32(resources.EtcdClusterSize)
	set.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
	set.Spec.PodManagementPolicy = appsv1.ParallelPodManagement
	set.Spec.ServiceName = resources.EtcdServiceName
	set.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			resources.AppLabelKey: name,
			"cluster":             data.Cluster.Name,
		},
	}

	set.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Name: name,
		Labels: map[string]string{
			resources.AppLabelKey: name,
			"cluster":             data.Cluster.Name,
		},
		Annotations: map[string]string{
			"prometheus.io/scrape": "true",
			"prometheus.io/path":   "/metrics",
			"prometheus.io/port":   "2379",
		},
	}

	// For migration purpose.
	// We switched from the etcd-operator to a simple etcd-StatefulSet. Therefore we need to migrate the data.
	var migrate bool
	_, err := data.ServiceLister.Services(data.Cluster.Status.NamespaceName).Get("etcd-cluster-client")
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
	} else {
		migrate = true
	}

	etcdStartCmd, err := getEtcdCommand(data.Cluster.Name, data.Cluster.Status.NamespaceName, migrate)
	if err != nil {
		return nil, err
	}
	set.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:                     name,
			Image:                    "quay.io/coreos/etcd:v3.2.20",
			ImagePullPolicy:          corev1.PullIfNotPresent,
			Command:                  etcdStartCmd,
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
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
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: defaultEtcdMemoryRequest,
					corev1.ResourceCPU:    defaultEtcdCPURequest,
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: defaultEtcdMemoryLimit,
					corev1.ResourceCPU:    defaultEtcdCPULimit,
				},
			},
			ReadinessProbe: &corev1.Probe{
				TimeoutSeconds:   1,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/health",
						Port: intstr.FromInt(2379),
					},
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "data",
					MountPath: "/var/run/etcd",
				},
			},
		},
	}

	// Make sure, we don't change size of existing pvc's
	diskSize := data.EtcdDiskSize
	if len(set.Spec.VolumeClaimTemplates) > 0 {
		if size, exists := set.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests[corev1.ResourceStorage]; exists {
			diskSize = size
		}
	}

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

	return set, nil
}

type commandTplData struct {
	ServiceName string
	Namespace   string
	Token       string
	DataDir     string
	Migrate     bool
}

func getEtcdCommand(name, namespace string, migrate bool) ([]string, error) {
	tpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(etcdStartCommandTpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse etcd command template: %v", err)
	}

	tplData := commandTplData{
		ServiceName: resources.EtcdServiceName,
		Token:       name,
		Namespace:   namespace,
		DataDir:     dataDir,
		Migrate:     migrate,
	}

	buf := bytes.Buffer{}
	if err := tpl.Execute(&buf, tplData); err != nil {
		return nil, err
	}

	return []string{
		"/bin/sh",
		"-ecx",
		buf.String(),
	}, nil
}

const (
	etcdStartCommandTpl = `ETCDCTL_API=3
MASTER_ENDPOINT="http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2379"

{{ if .Migrate }}
# If we're already initialized
if [ -d "{{ .DataDir }}" ]; then
    INITIAL_STATE="existing"
    INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-1=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-2=http://etcd-2.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"
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
        INITIAL_STATE="new"
        INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"
    fi

    if [ "${POD_NAME}" = "etcd-1" ]; then
        echo "i'm etcd-1. I join as new member as soon as etcd-0 comes up"
        etcdctl --endpoints ${MASTER_ENDPOINT} member add etcd-1 --peer-urls=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2379
        INITIAL_STATE="existing"
        INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-1=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"
    fi

    if [ "${POD_NAME}" = "etcd-2" ]; then
        echo "i'm etcd-2. I join as new member as soon as we have 2 existing & healthy members"
        until etcdctl --endpoints ${MASTER_ENDPOINT} member list | grep -q etcd-1; do sleep 1; echo "Waiting for etcd-1"; done
        INITIAL_STATE="existing"
        INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-1=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-2=http://etcd-2.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"
    fi
fi

{{ else }}
INITIAL_STATE="new"
INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-1=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-2=http://etcd-2.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"
{{ end }}

echo ${INITIAL_STATE}
echo ${INITIAL_CLUSTER}

exec /usr/local/bin/etcd \
    --name=${POD_NAME} \
    --data-dir="{{ .DataDir }}" \
    --heartbeat-interval=500 \
    --election-timeout=5000 \
    --initial-cluster=${INITIAL_CLUSTER} \
    --initial-cluster-token="{{ .Token }}" \
    --initial-cluster-state=${INITIAL_STATE} \
    --advertise-client-urls http://${POD_NAME}.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2379 \
    --listen-client-urls http://0.0.0.0:2379 \
    --listen-peer-urls http://0.0.0.0:2380
`
)
