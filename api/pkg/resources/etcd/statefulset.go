package etcd

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	etcdDiskSize = resource.MustParse("5Gi")
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

	set.Spec.Replicas = resources.Int32(3)
	set.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
	set.Spec.PodManagementPolicy = appsv1.ParallelPodManagement
	set.Spec.ServiceName = resources.EtcdServiceName
	set.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app":     "etcd",
			"cluster": data.Cluster.Name,
		},
	}

	set.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Name: "etcd",
		Labels: map[string]string{
			"app":     "etcd",
			"cluster": data.Cluster.Name,
		},
	}

	etcdCmd, err := getEtcdCommand(data)
	if err != nil {
		return nil, err
	}
	set.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:                     "etcd",
			Image:                    "quay.io/coreos/etcd:v3.3.5",
			Command:                  etcdCmd,
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
			ReadinessProbe: &corev1.Probe{
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
					Requests: corev1.ResourceList{corev1.ResourceStorage: etcdDiskSize},
				},
			},
		},
	}

	return set, nil
}

func getEtcdCommand(data *resources.TemplateData) ([]string, error) {
	tpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(etcdCommandTpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse etcd command template: %v", err)
	}

	tplData := struct {
		ServiceName string
		Namespace   string
		Token       string
	}{
		ServiceName: resources.EtcdServiceName,
		Token:       data.Cluster.Name,
		Namespace:   data.Cluster.Status.NamespaceName,
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

const etcdCommandTpl = `/usr/local/bin/etcd \
--name=$(POD_NAME) \
--data-dir="/var/run/etcd" \
--heartbeat-interval=500 \
--election-timeout=5000 \
--initial-cluster="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-1=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-2=http://etcd-2.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380" \
--initial-cluster-token="{{ .Token }}" \
--initial-cluster-state=new \
--advertise-client-urls http://${POD_NAME}.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2379 \
--listen-client-urls http://0.0.0.0:2379 \
--listen-peer-urls http://0.0.0.0:2380
`
