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
	"fmt"
	"strconv"
	"strings"

	semverlib "github.com/Masterminds/semver/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	name    = "etcd"
	dataDir = "/var/run/etcd/pod_$(POD_NAME)/"

	memberListPattern = "etcd-%d=http://etcd-%d.%s.%s.svc.cluster.local:2380"
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

type etcdStatefulSetReconcilerData interface {
	Cluster() *kubermaticv1.Cluster
	RewriteImage(string) (string, error)
	EtcdDiskSize() resource.Quantity
	EtcdLauncherImage() string
	EtcdLauncherTag() string
	GetClusterRef() metav1.OwnerReference
	SupportsFailureDomainZoneAntiAffinity() bool
}

// StatefulSetReconciler returns the function to reconcile the etcd StatefulSet.
func StatefulSetReconciler(data etcdStatefulSetReconcilerData, enableDataCorruptionChecks, enableTLSOnly bool, quotaBackendGB int64) reconciling.NamedStatefulSetReconcilerFactory {
	return func() (string, reconciling.StatefulSetReconciler) {
		return resources.EtcdStatefulSetName, func(set *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
			replicas := computeReplicas(data, set)
			imageTag := ImageTag(data.Cluster())

			imageTagVersion, err := semverlib.NewVersion(imageTag)
			if err != nil {
				return nil, fmt.Errorf("failed to parse etcd image tag: %w", err)
			}

			etcdConstraint, err := semverlib.NewConstraint(">= 3.5.0, < 3.6.0")
			if err != nil {
				return nil, fmt.Errorf("failed to parse etcd constraint: %w", err)
			}

			// enable initial and periodic etcd data corruption checks by default if running etcd 3.5.
			// The etcd team has recommended to enable this feature for etcd 3.5 due to data consistency issues.
			// Reference: https://groups.google.com/a/kubernetes.io/g/dev/c/B7gJs88XtQc/m/rSgNOzV2BwAJ
			if ok := etcdConstraint.Check(imageTagVersion); ok {
				enableDataCorruptionChecks = true
			}

			set.Spec.Replicas = resources.Int32(replicas)
			set.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
			set.Spec.PodManagementPolicy = appsv1.ParallelPodManagement
			set.Spec.ServiceName = resources.EtcdServiceName
			set.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			baseLabels := GetBasePodLabels(data.Cluster())
			set.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			set.Spec.Template.Name = name
			set.Spec.Template.Spec.ServiceAccountName = rbac.EtcdLauncherServiceAccountName

			kubernetes.EnsureAnnotations(&set.Spec.Template, map[string]string{
				// NB: We purposefully do not want to use the cluster-last-restart annotation here to
				// restart etcd, as that would lead to multiple complete restarts during an etcd restore.
				// resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],

				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: "launcher",
			})

			etcdEnv := []corev1.EnvVar{
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
			}

			etcdPorts := []corev1.ContainerPort{
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
			}

			launcherEnabled := data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher]
			if launcherEnabled {
				set.Spec.Template.Spec.InitContainers = []corev1.Container{
					{
						Name:            "etcd-launcher-init",
						Image:           fmt.Sprintf("%s:%s", data.EtcdLauncherImage(), data.EtcdLauncherTag()),
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

				var etcdEndpoints []string
				for i := range 3 {
					endpoint := fmt.Sprintf(
						"https://etcd-%d.%s.%s.svc.cluster.local:2379",
						i,
						resources.EtcdServiceName,
						data.Cluster().Status.NamespaceName,
					)
					etcdEndpoints = append(etcdEndpoints, endpoint)
				}
				etcdEnv = append(etcdEnv, corev1.EnvVar{Name: "ETCDCTL_ENDPOINTS", Value: strings.Join(etcdEndpoints, ",")})

				etcdPorts = append(etcdPorts, corev1.ContainerPort{
					ContainerPort: 2381,
					Protocol:      corev1.ProtocolTCP,
					Name:          "peer-tls",
				})

				kubernetes.EnsureAnnotations(&set.Spec.Template, map[string]string{
					resources.EtcdTLSEnabledAnnotation: "",
				})

				if enableTLSOnly {
					etcdEnv = append(etcdEnv, corev1.EnvVar{Name: "PEER_TLS_MODE", Value: "strict"})
				}
			} else {
				endpoints := []string{}

				for i := range 3 {
					endpoints = append(endpoints, fmt.Sprintf(memberListPattern, i, i, resources.EtcdServiceName, data.Cluster().Status.NamespaceName))
				}

				etcdEnv = append(etcdEnv, corev1.EnvVar{Name: "MASTER_ENDPOINT", Value: fmt.Sprintf("https://etcd-0.%s.%s.svc.cluster.local:2379", resources.EtcdServiceName, data.Cluster().Status.NamespaceName)})
				etcdEnv = append(etcdEnv, corev1.EnvVar{Name: "INITIAL_CLUSTER", Value: strings.Join(endpoints, ",")})
			}

			set.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            resources.EtcdStatefulSetName,
					Image:           registry.Must(data.RewriteImage(resources.RegistryK8S + "/etcd:" + imageTag + "-0")),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         getEtcdCommand(data.Cluster(), enableDataCorruptionChecks, launcherEnabled, quotaBackendGB),
					Env:             etcdEnv,
					Ports:           etcdPorts,
					ReadinessProbe: &corev1.Probe{
						TimeoutSeconds:      10,
						PeriodSeconds:       15,
						SuccessThreshold:    1,
						FailureThreshold:    3,
						InitialDelaySeconds: 5,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/readyz",
								Port:   intstr.FromInt(2378),
								Scheme: corev1.URISchemeHTTP,
							},
						},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/livez",
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
					// timing info calculated based on
					// https://github.com/kubernetes/kubernetes/blob/master/cmd/kubeadm/app/util/staticpod/utils.go#L254-L265
					StartupProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/readyz",
								Port:   intstr.FromInt(2378),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 10,
						FailureThreshold:    16,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
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

			set.Spec.Template.Spec.Tolerations = data.Cluster().Spec.ComponentsOverride.Etcd.Tolerations

			err = resources.SetResourceRequirements(set.Spec.Template.Spec.Containers, defaultResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), set.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			hostAntiAffinityType := data.Cluster().Spec.ComponentsOverride.Etcd.HostAntiAffinity
			set.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(resources.EtcdStatefulSetName, hostAntiAffinityType)

			if data.SupportsFailureDomainZoneAntiAffinity() {
				zoneAntiAffinityType := data.Cluster().Spec.ComponentsOverride.Etcd.ZoneAntiAffinity
				failureDomainZoneAntiAffinity := resources.FailureDomainZoneAntiAffinity(resources.EtcdStatefulSetName, zoneAntiAffinityType)
				set.Spec.Template.Spec.Affinity = resources.MergeAffinities(set.Spec.Template.Spec.Affinity, failureDomainZoneAntiAffinity)
			}

			set.Spec.Template.Spec.NodeSelector = data.Cluster().Spec.ComponentsOverride.Etcd.NodeSelector

			set.Spec.Template.Spec.Volumes = getVolumes()

			// Make sure we don't change volume claim template of existing sts
			if len(set.Spec.VolumeClaimTemplates) == 0 {
				storageClass := data.Cluster().Spec.ComponentsOverride.Etcd.StorageClass
				if storageClass == "" {
					storageClass = "kubermatic-fast"
				}
				diskSize := data.Cluster().Spec.ComponentsOverride.Etcd.DiskSize
				if diskSize == nil {
					d := data.EtcdDiskSize()
					diskSize = &d
				}
				set.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "data",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							StorageClassName: resources.String(storageClass),
							AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{corev1.ResourceStorage: *diskSize},
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

func GetBasePodLabels(cluster *kubermaticv1.Cluster) map[string]string {
	additionalLabels := map[string]string{
		"cluster": cluster.Name,
	}
	return resources.BaseAppLabels(resources.EtcdStatefulSetName, additionalLabels)
}

// ImageTag returns the correct etcd image tag for a given Cluster. Note that this tag does not
// contain the "-0" suffix that the registry.k8s.io images have appended to them. This is because
// semver comparisons then fail further up in the code and it's simpler to treat the "-0" suffix
// as not part of the etcd tag itself.
// TODO: Other functions use this function, switch them to getLauncherImage.
func ImageTag(c *kubermaticv1.Cluster) string {
	// most other control plane parts refer to the controller-manager's version, which
	// during updates lacks behind the apiserver by one minor version; this is so that
	// also external components like the kubernetes dashboard or external ccms wait for
	// the new apiserver to be ready; etcd however is different and gets updated together
	// with the apiserver.
	// As of now, all supported Kubernetes versions use the same etcd release, but the
	// comment above is left as a reminder in case future versions switches will be needed
	// again.
	//
	// See the SupportedEtcdVersion variable in
	// https://github.com/kubernetes/kubernetes/blob/master/cmd/kubeadm/app/constants/constants.go
	// for an overview.

	// if c.Status.Versions.Apiserver.LessThan(semver.NewSemverOrDie("1.22.0")) {
	// 	return "v3.4.3"
	// }

	return "3.5.21"
}

func computeReplicas(data etcdStatefulSetReconcilerData, set *appsv1.StatefulSet) int32 {
	if !data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] {
		return kubermaticv1.DefaultEtcdClusterSize
	}
	etcdClusterSize := getClusterSize(data.Cluster().Spec.ComponentsOverride.Etcd)
	if set.Spec.Replicas == nil { // new replicaset
		return etcdClusterSize
	}
	replicas := *set.Spec.Replicas
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

func getClusterSize(settings kubermaticv1.EtcdStatefulSetSettings) int32 {
	if settings.ClusterSize == nil {
		return kubermaticv1.DefaultEtcdClusterSize
	}
	if *settings.ClusterSize < kubermaticv1.MinEtcdClusterSize {
		return kubermaticv1.MinEtcdClusterSize
	}
	if *settings.ClusterSize > kubermaticv1.MaxEtcdClusterSize {
		return kubermaticv1.MaxEtcdClusterSize
	}
	return *settings.ClusterSize
}

func getEtcdCommand(cluster *kubermaticv1.Cluster, enableCorruptionCheck, launcherEnabled bool, quotaBackendGB int64) []string {
	if launcherEnabled {
		command := []string{"/opt/bin/etcd-launcher",
			"run",
			"--cluster", cluster.Name,
			"--pod-name", "$(POD_NAME)",
			"--pod-ip", "$(POD_IP)",
			"--api-version", "$(ETCDCTL_API)",
			"--token", "$(TOKEN)",
		}

		if enableCorruptionCheck {
			command = append(command, "--enable-corruption-check")
		}

		if quotaBackendGB > 0 {
			command = append(command, "--quota-backend-gb", strconv.FormatInt(quotaBackendGB, 10))
		}

		return command
	}

	// construct command for "plain" etcd usage.

	command := []string{
		"/usr/local/bin/etcd",
		"--name",
		"$(POD_NAME)",
		"--data-dir",
		dataDir,
		"--initial-cluster",
		"$(INITIAL_CLUSTER)",
		"--initial-cluster-token",
		cluster.Name,
		"--initial-cluster-state",
		"new",
		"--advertise-client-urls",
		fmt.Sprintf("https://$(POD_NAME).%s.%s.svc.cluster.local:2379,https://$(POD_IP):2379", resources.EtcdServiceName, cluster.Status.NamespaceName),
		"--listen-client-urls",
		"https://$(POD_IP):2379,https://127.0.0.1:2379",
		"--listen-peer-urls",
		"http://$(POD_IP):2380",
		"--listen-metrics-urls",
		"http://$(POD_IP):2378,http://127.0.0.1:2378",
		"--initial-advertise-peer-urls",
		fmt.Sprintf("http://$(POD_NAME).%s.%s.svc.cluster.local:2380", resources.EtcdServiceName, cluster.Status.NamespaceName),
		"--trusted-ca-file",
		"/etc/etcd/pki/ca/ca.crt",
		"--client-cert-auth",
		"--cert-file",
		"/etc/etcd/pki/tls/etcd-tls.crt",
		"--key-file",
		"/etc/etcd/pki/tls/etcd-tls.key",
		"--auto-compaction-retention",
		"8",
	}

	if enableCorruptionCheck {
		command = append(command, "--experimental-initial-corrupt-check")
		command = append(command, "--experimental-corrupt-check-time", "240m")
	}

	if quotaBackendGB > 0 {
		bytes, overflow := resources.ConvertGBToBytes(uint64(quotaBackendGB))
		if !overflow {
			command = append(command, "--quota-backend-bytes", strconv.FormatUint(bytes, 10))
		}
	}

	return command
}
