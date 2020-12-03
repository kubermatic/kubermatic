/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcd

import (
	"bytes"
	"fmt"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
)

const (
	name = "etcd"

	dataDir = "/var/run/etcd/pod_${POD_NAME}/"
	// ImageTag defines the image tag to use for the etcd image
	etcdImageTagV33 = "v3.3.18"
	etcdImageTagV34 = "v3.4.3"
)

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		name: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("2Gi"),
				corev1.ResourceCPU:    resource.MustParse("2"),
			},
		},
	}
)

type etcdStatefulSetCreatorData interface {
	Cluster() *kubermaticv1.Cluster
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	ImageRegistry(string) string
	EtcdDiskSize() resource.Quantity
	EtcdLauncherImage() string
	EtcdLauncherTag() string
	GetClusterRef() metav1.OwnerReference
	SupportsFailureDomainZoneAntiAffinity() bool
}

// StatefulSetCreator returns the function to reconcile the etcd StatefulSet
func StatefulSetCreator(data etcdStatefulSetCreatorData, enableDataCorruptionChecks bool) reconciling.NamedStatefulSetCreatorGetter {
	return func() (string, reconciling.StatefulSetCreator) {
		return resources.EtcdStatefulSetName, func(set *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {

			replicas := computeReplicas(data, set)
			set.Name = resources.EtcdStatefulSetName
			set.Spec.Replicas = resources.Int32(int32(replicas))
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
			set.Spec.Template.Spec.ServiceAccountName = rbac.EtcdLauncherServiceAccountName

			launcherEnabled := data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher]
			if launcherEnabled {
				set.Spec.Template.Spec.InitContainers = []corev1.Container{
					{
						Name:            "etcd-launcher-init",
						Image:           data.EtcdLauncherImage() + ":" + data.EtcdLauncherTag(),
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         []string{"/bin/cp", "/etcd-launcher", "/opt/bin/"},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "launcher",
								MountPath: "/opt/bin/",
							},
						},
					},
				}
			}
			etcdStartCmd, err := getEtcdCommand(data.Cluster().Name, data.Cluster().Status.NamespaceName, enableDataCorruptionChecks, launcherEnabled)
			if err != nil {
				return nil, err
			}
			set.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name: resources.EtcdStatefulSetName,

					Image:           data.ImageRegistry(resources.RegistryGCR) + "/etcd-development/etcd:" + ImageTag(data.Cluster()),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         etcdStartCmd,
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
							Name: "NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									APIVersion: "v1",
									FieldPath:  "metadata.namespace",
								},
							},
						},
						{
							Name:  "TOKEN",
							Value: data.Cluster().Name,
						},
						{
							Name:  "ETCD_CLUSTER_SIZE",
							Value: strconv.Itoa(replicas),
						},
						{
							Name:  "ENABLE_CORRUPTION_CHECK",
							Value: strconv.FormatBool(enableDataCorruptionChecks),
						},
						{
							Name:  "ETCDCTL_API",
							Value: "3",
						},
						{
							Name:  "ETCDCTL_CACERT",
							Value: resources.EtcdTrustedCAFile,
						},
						{
							Name:  "ETCDCTL_CERT",
							Value: resources.EtcdClientCertFile,
						},
						{
							Name:  "ETCDCTL_KEY",
							Value: resources.EtcdClientKeyFile,
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
						TimeoutSeconds:      10,
						PeriodSeconds:       15,
						SuccessThreshold:    1,
						FailureThreshold:    3,
						InitialDelaySeconds: 5,
						Handler: corev1.Handler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"/usr/local/bin/etcdctl",
									"--command-timeout", "10s",
									"endpoint", "health",
								},
							},
						},
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/health",
								Port:   intstr.FromInt(2378),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 5,
						FailureThreshold:    3,
						PeriodSeconds:       30,
						SuccessThreshold:    1,
						TimeoutSeconds:      10,
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
						{
							Name:      "launcher",
							MountPath: "/opt/bin/",
						},
					},
				},
			}
			err = resources.SetResourceRequirements(set.Spec.Template.Spec.Containers, defaultResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), set.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
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
		{
			Name: "launcher",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func getBasePodLabels(cluster *kubermaticv1.Cluster) map[string]string {
	additionalLabels := map[string]string{
		"cluster": cluster.Name,
	}
	return resources.BaseAppLabels(resources.EtcdStatefulSetName, additionalLabels)
}

// ImageTag returns the correct etcd image tag for a given Cluster
// TODO: Other functions use this function, switch them to getLauncherImage
func ImageTag(c *kubermaticv1.Cluster) string {
	if c.IsOpenshift() || c.Spec.Version.Minor() < 17 {
		return etcdImageTagV33
	}
	return etcdImageTagV34
}

func computeReplicas(data etcdStatefulSetCreatorData, set *appsv1.StatefulSet) int {
	if !data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] {
		return kubermaticv1.DefaultEtcdClusterSize
	}
	etcdClusterSize := data.Cluster().Spec.ComponentsOverride.Etcd.ClusterSize
	// handle existing clusters that don't have a configured size
	if etcdClusterSize < kubermaticv1.DefaultEtcdClusterSize {
		klog.V(2).Infof("etcdClusterSize [%d] is smaller than DefaultEtcdClusterSize [%d]. Falling back to DefaultEtcdClusterSize", etcdClusterSize, kubermaticv1.DefaultEtcdClusterSize)
		etcdClusterSize = kubermaticv1.DefaultEtcdClusterSize
	}
	if etcdClusterSize > kubermaticv1.MaxEtcdClusterSize {
		klog.V(2).Infof("etcdClusterSize [%d] is larger than MaxEtcdClusterSize [%d]. Falling back to MaxEtcdClusterSize", etcdClusterSize, kubermaticv1.MaxEtcdClusterSize)
		etcdClusterSize = kubermaticv1.MaxEtcdClusterSize
	}
	if set.Spec.Replicas == nil { // new replicaset
		return etcdClusterSize
	}
	replicas := int(*set.Spec.Replicas)
	// at required size. do nothing
	if etcdClusterSize == replicas {
		return replicas
	}
	isEtcdHealthy := data.Cluster().Status.ExtendedHealth.Etcd == kubermaticv1.HealthStatusUp
	if isEtcdHealthy { // no scaling until we are healthy

		if etcdClusterSize > replicas {
			return replicas + 1
		}
		return replicas - 1
	}
	return replicas
}

type commandTplData struct {
	ServiceName           string
	Namespace             string
	Token                 string
	DataDir               string
	Migrate               bool
	EnableCorruptionCheck bool
}

func getEtcdCommand(name, namespace string, enableCorruptionCheck, launcherEnabled bool) ([]string, error) {
	if launcherEnabled {
		command := []string{"/opt/bin/etcd-launcher",
			"-namespace", "$(NAMESPACE)",
			"-etcd-cluster-size", "$(ETCD_CLUSTER_SIZE)",
			"-pod-name", "$(POD_NAME)",
			"-pod-ip", "$(POD_IP)",
			"-api-version", "$(ETCDCTL_API)",
			"-token", "$(TOKEN)"}
		if enableCorruptionCheck {
			command = append(command, "-enable-corruption-check")
		}
		return command, nil
	}

	tpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(etcdStartCommandTpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse etcd command template: %v", err)
	}

	tplData := commandTplData{
		ServiceName:           resources.EtcdServiceName,
		Token:                 name,
		Namespace:             namespace,
		DataDir:               dataDir,
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

export INITIAL_STATE="new"
export INITIAL_CLUSTER="etcd-0=http://etcd-0.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-1=http://etcd-1.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380,etcd-2=http://etcd-2.{{ .ServiceName }}.{{ .Namespace }}.svc.cluster.local:2380"

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
    --listen-metrics-urls "http://${POD_IP}:2378,http://127.0.0.1:2378" \
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
