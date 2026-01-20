/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package konnectivity

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/konnectivity"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

var (
	defResourceRequirements = map[string]*corev1.ResourceRequirements{
		resources.KonnectivityAgentContainer: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("10Mi"),
				corev1.ResourceCPU:    resource.MustParse("10m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("100Mi"),
				corev1.ResourceCPU:    resource.MustParse("2"),
			},
		},
	}
)

// DeploymentReconciler returns function to reconcile konnectivity agents deployment in user cluster.
func DeploymentReconciler(
	clusterVersion semver.Semver,
	cluster *kubermaticv1.Cluster,
	kServerHost string,
	kServerPort int,
	kKeepaliveTime string,
	imageRewriter registry.ImageRewriter,
	kResourcesOverrides map[string]*corev1.ResourceRequirements,
) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.KonnectivityDeploymentName, func(ds *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := resources.BaseAppLabels(resources.KonnectivityDeploymentName, nil)
			ds.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
			ds.Spec.Template.Labels = labels

			replicas := int32(2)
			ds.Spec.Replicas = &replicas

			maxUnavailable := intstr.FromInt(1)
			maxSurge := intstr.FromString("25%")
			ds.Spec.Strategy = appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
			}

			args := getArgs(cluster, kServerHost, kKeepaliveTime, kServerPort)
			ds.Spec.Template.Spec.PriorityClassName = "system-cluster-critical"
			ds.Spec.Template.Spec.ServiceAccountName = resources.KonnectivityServiceAccountName
			ds.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name: resources.KonnectivityAgentContainer,
					Image: registry.Must(imageRewriter(
						fmt.Sprintf("%s/kas-network-proxy/proxy-agent:%s",
							resources.RegistryK8S,
							konnectivity.NetworkProxyVersion(clusterVersion),
						),
					)),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/proxy-agent"},
					Args:            args,
					Resources:       corev1.ResourceRequirements{},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.KonnectivityAgentToken,
							MountPath: "/var/run/secrets/tokens",
						},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/healthz",
								Port: intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: 8134,
								},
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 15,
						TimeoutSeconds:      15,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						FailureThreshold:    3,
					},
				},
			}
			ds.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: resources.KonnectivityAgentToken,
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{
								{
									ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
										Audience:          resources.KonnectivityClusterRoleBindingUsername,
										Path:              resources.KonnectivityAgentToken,
										ExpirationSeconds: ptr.To[int64](3600),
									},
								},
							},
							DefaultMode: ptr.To[int32](420),
						},
					},
				},
			}

			ds.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			err := resources.SetResourceRequirements(ds.Spec.Template.Spec.Containers, defResourceRequirements, kResourcesOverrides, ds.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			ds.Spec.Template.Spec.Affinity = &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							Weight: 10,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: resources.BaseAppLabels(resources.KonnectivityDeploymentName, nil),
								},
								TopologyKey: resources.TopologyKeyHostname,
							},
						},
					},
				},
			}
			return ds, nil
		}
	}
}

func getArgs(cluster *kubermaticv1.Cluster, kServerHost, kKeepaliveTime string, kServerPort int) []string {
	args := []string{
		"--logtostderr=true",
		"-v=3",
		"--sync-forever=true",
		"--ca-cert=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		fmt.Sprintf("--proxy-server-host=%s", kServerHost),
		fmt.Sprintf("--proxy-server-port=%d", kServerPort),
		"--admin-server-port=8133",
		"--health-server-port=8134",
		fmt.Sprintf("--service-account-token-path=/var/run/secrets/tokens/%s", resources.KonnectivityAgentToken),
		// TODO rastislavs: use "--agent-identifiers=ipv4=$(HOST_IP)" with "--proxy-strategies=destHost,default"
		// once the upstream issue is resolved: https://github.com/kubernetes-sigs/apiserver-network-proxy/issues/261
		fmt.Sprintf("--keepalive-time=%s", kKeepaliveTime),
	}

	clusterArgs := []string{}
	if cluster != nil {
		clusterArgs = cluster.Spec.ComponentsOverride.KonnectivityProxy.Args
	}

	if !konnectivity.HasArgWithPrefix(clusterArgs, "--xfr-channel-size") {
		args = append(args, fmt.Sprintf("--xfr-channel-size=%d", kubermaticv1.DefaultKonnectivityXfrChannelSize))
	}

	args = append(args, clusterArgs...)
	return args
}

// PodDisruptionBudgetReconciler returns a func to create/update the Konnectivity agent's PodDisruptionBudget.
func PodDisruptionBudgetReconciler() reconciling.NamedPodDisruptionBudgetReconcilerFactory {
	return func() (string, reconciling.PodDisruptionBudgetReconciler) {
		return resources.KonnectivityPodDisruptionBudgetName, func(pdb *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
			minAvailable := intstr.FromInt(1)
			pdb.Spec = policyv1.PodDisruptionBudgetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: resources.BaseAppLabels(resources.KonnectivityDeploymentName, nil),
				},
				MinAvailable: &minAvailable,
			}
			return pdb, nil
		}
	}
}
