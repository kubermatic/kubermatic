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

package nodeportproxy

import (
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	name               = "nodeport-proxy"
	imageName          = "kubermatic/nodeport-proxy"
	envoyAppLabelValue = resources.NodePortProxyEnvoyDeploymentName

	EnvoyVersion = "distroless-v1.37.0"

	// NodePortProxyExposeNamespacedAnnotationKey is the annotation key used to indicate that
	// a service should be exposed by the namespaced NodeportProxy instance.
	// We use it when clusters get exposed via a LoadBalancer, to allow reusing that LoadBalancer
	// for the kube-apiserver.
	NodePortProxyExposeNamespacedAnnotationKey = "nodeport-proxy.k8s.io/expose-namespaced"
	DefaultExposeAnnotationKey                 = "nodeport-proxy.k8s.io/expose"
	// PortHostMappingAnnotationKey contains the mapping between the port to be
	// exposed and the hostname, this is only used when the ExposeType is
	// SNIType.
	PortHostMappingAnnotationKey = "nodeport-proxy.k8s.io/port-mapping"

	loadBalancerSourceRangesAnnotationKey = "service.beta.kubernetes.io/load-balancer-source-ranges"
)

// ExposeType defines the strategy used to expose the service.
type ExposeType int

const (
	// NodePortType is the default ExposeType which creates a listener for each
	// NodePort.
	NodePortType ExposeType = iota
	// SNIType configures Envoy to route TLS streams based on SNI
	// without terminating them.
	SNIType
	// TunnelingType configures Envoy to terminate the tunnel and stream the
	// data to the destination.
	// The only supported tunneling technique at the moment in HTTP/2 Connect.
	TunnelingType
)

// exposeTypeStrings contains the string representation of the ExposeTypes.
var exposeTypeStrings = [...]string{"NodePort", "SNI", "Tunneling"}

// ExposeTypeFromString returns the ExposeType which string representation
// corresponds to the input string, and a boolean indicating whether the
// corresponding ExposeType was found or not.
func ExposeTypeFromString(s string) (ExposeType, bool) {
	switch s {
	case exposeTypeStrings[NodePortType]:
		return NodePortType, true
	case exposeTypeStrings[SNIType]:
		return SNIType, true
	case exposeTypeStrings[TunnelingType]:
		return TunnelingType, true
	default:
		return NodePortType, false
	}
}

// String returns the string representation of the ExposeType.
func (e ExposeType) String() string {
	return exposeTypeStrings[e]
}

type ExposeTypes map[ExposeType]sets.Empty

func NewExposeTypes(exposeTypes ...ExposeType) ExposeTypes {
	ets := ExposeTypes{}
	for _, et := range exposeTypes {
		ets[et] = sets.Empty{}
	}
	return ets
}

func (e ExposeTypes) Has(item ExposeType) bool {
	_, contained := e[item]
	return contained
}

func (e ExposeTypes) Insert(item ExposeType) {
	e[item] = sets.Empty{}
}

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		"envoy-manager": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("32Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("48Mi"),
			},
		},
		resources.NodePortProxyEnvoyContainerName: {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("32Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
		},
		"lb-updater": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("32Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("32Mi"),
			},
		},
	}
)

type nodePortProxyData interface {
	RewriteImage(string) (string, error)
	NodePortProxyTag() string
	Cluster() *kubermaticv1.Cluster
	Seed() *kubermaticv1.Seed
	SupportsFailureDomainZoneAntiAffinity() bool
}

func ServiceAccountReconciler() (string, reconciling.ServiceAccountReconciler) {
	return name, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		return sa, nil
	}
}

func RoleReconciler() (string, reconciling.RoleReconciler) {
	return name, func(r *rbacv1.Role) (*rbacv1.Role, error) {
		r.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services"},
				Verbs:     []string{"list", "get", "watch"},
			},
			{
				APIGroups: []string{"discovery.k8s.io"},
				Resources: []string{"endpointslices"},
				Verbs:     []string{"list", "get", "watch"},
			},
			{
				APIGroups:     []string{""},
				Resources:     []string{"services"},
				ResourceNames: []string{resources.FrontLoadBalancerServiceName},
				Verbs:         []string{"update"},
			},
		}
		return r, nil
	}
}

func RoleBindingReconciler() (string, reconciling.RoleBindingReconciler) {
	return name, func(r *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
		r.Subjects = []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: name,
			},
		}
		r.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		}
		return r, nil
	}
}

func DeploymentEnvoyReconciler(data nodePortProxyData, versions kubermatic.Versions) reconciling.NamedDeploymentReconcilerFactory {
	volumeMountNameEnvoyConfig := "envoy-config"
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.NodePortProxyEnvoyDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(envoyAppLabelValue, nil)
			kubernetes.EnsureLabels(d, baseLabels)

			d.Spec.Replicas = resources.Int32(2)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			kubernetes.EnsureLabels(&d.Spec.Template, baseLabels)
			kubernetes.EnsureAnnotations(&d.Spec.Template, map[string]string{
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: volumeMountNameEnvoyConfig,
			})

			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: resources.ImagePullSecretName},
			}

			d.Spec.Template.Spec.InitContainers = []corev1.Container{
				{
					Name:  "copy-envoy-config",
					Image: registry.Must(data.RewriteImage(fmt.Sprintf("%s/%s:%s", resources.RegistryQuay, imageName, data.NodePortProxyTag()))),
					Command: []string{
						"/bin/cp",
						"/envoy.yaml",
						"/etc/envoy/envoy.yaml",
					},
					VolumeMounts: []corev1.VolumeMount{{
						Name:      volumeMountNameEnvoyConfig,
						MountPath: "/etc/envoy",
					}},
				},
			}

			seed := data.Seed()

			d.Spec.Template.Spec.Containers = []corev1.Container{{
				Name:  "envoy-manager",
				Image: registry.Must(data.RewriteImage(fmt.Sprintf("%s/%s:%s", resources.RegistryQuay, imageName, data.NodePortProxyTag()))),
				Command: []string{"/envoy-manager",
					"-listen-address=:8001",
					"-envoy-node-name=$(PODNAME)",
					"-envoy-admin-port=9001",
					"-envoy-stats-port=8002",
					"-expose-annotation-key=" + NodePortProxyExposeNamespacedAnnotationKey,
					"-namespace=$(PODNAMESPACE)"},
				Env: []corev1.EnvVar{
					{
						Name: "PODNAMESPACE",
						ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						}},
					},
					{
						Name: "PODNAME",
						ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						}},
					},
				},
			}, {
				Name:  resources.NodePortProxyEnvoyContainerName,
				Image: registry.Must(data.RewriteImage(fmt.Sprintf("%s:%s", seed.Spec.NodeportProxy.Envoy.DockerRepository, EnvoyVersion))),
				Command: []string{
					"/usr/local/bin/envoy",
					"-c",
					"/etc/envoy/envoy.yaml",
					"--service-cluster",
					"kube-cluster",
					"--service-node",
					"$(PODNAME)",
				},
				Env: []corev1.EnvVar{
					{
						Name: "PODNAME",
						ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						}},
					},
				},
				Lifecycle: &corev1.Lifecycle{
					PreStop: &corev1.LifecycleHandler{
						Exec: &corev1.ExecAction{
							Command: []string{
								"wget",
								"-q0",
								"http://127.0.0.1:9001/healthcheck/fail",
							},
						},
					},
				},
				ReadinessProbe: &corev1.Probe{
					FailureThreshold: 3,
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path:   "healthz",
							Port:   intstr.FromInt(8002),
							Scheme: corev1.URISchemeHTTP,
						},
					},
					PeriodSeconds:    3,
					SuccessThreshold: 1,
					TimeoutSeconds:   1,
				},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      volumeMountNameEnvoyConfig,
					MountPath: "/etc/envoy",
				}},
			}}
			err := resources.SetResourceRequirements(d.Spec.Template.Spec.Containers, defaultResourceRequirements, resources.GetOverrides(data.Cluster().Spec.ComponentsOverride), d.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			d.Spec.Template.Spec.Affinity = resources.HostnameAntiAffinity(envoyAppLabelValue, kubermaticv1.AntiAffinityTypePreferred)
			if data.SupportsFailureDomainZoneAntiAffinity() {
				failureDomainZoneAntiAffinity := resources.FailureDomainZoneAntiAffinity(envoyAppLabelValue, kubermaticv1.AntiAffinityTypePreferred)
				d.Spec.Template.Spec.Affinity = resources.MergeAffinities(d.Spec.Template.Spec.Affinity, failureDomainZoneAntiAffinity)
			}

			d.Spec.Template.Spec.Volumes = []corev1.Volume{{
				Name: volumeMountNameEnvoyConfig,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}}
			d.Spec.Template.Spec.ServiceAccountName = "nodeport-proxy"
			return d, nil
		}
	}
}

func DeploymentLBUpdaterReconciler(data nodePortProxyData) reconciling.NamedDeploymentReconcilerFactory {
	deploymentName := fmt.Sprintf("%s-lb-updater", name)
	return func() (string, reconciling.DeploymentReconciler) {
		return deploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Name = deploymentName
			d.Labels = resources.BaseAppLabels(deploymentName, nil)
			d.Spec.Replicas = resources.Int32(1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(deploymentName, nil)}
			d.Spec.Template.Labels = resources.BaseAppLabels(deploymentName, nil)
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: resources.ImagePullSecretName},
			}

			d.Spec.Template.Spec.Containers = []corev1.Container{{
				Name: "lb-updater",
				Command: []string{
					"/lb-updater",
					"-lb-namespace=$(MY_NAMESPACE)",
					"-lb-name=" + resources.FrontLoadBalancerServiceName,
					"-expose-annotation-key=" + NodePortProxyExposeNamespacedAnnotationKey,
					"-namespaced=true",
				},
				Image: registry.Must(data.RewriteImage(fmt.Sprintf("%s/%s:%s", resources.RegistryQuay, imageName, data.NodePortProxyTag()))),
				Env: []corev1.EnvVar{{
					Name: "MY_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					}},
				}},
			}}
			err := resources.SetResourceRequirements(d.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, d.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}
			d.Spec.Template.Spec.ServiceAccountName = "nodeport-proxy"

			return d, nil
		}
	}
}

func PodDisruptionBudgetReconciler() reconciling.NamedPodDisruptionBudgetReconcilerFactory {
	maxUnavailable := intstr.FromInt(1)
	return func() (string, reconciling.PodDisruptionBudgetReconciler) {
		return name + "-envoy", func(pdb *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
			pdb.Spec.MaxUnavailable = &maxUnavailable
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(envoyAppLabelValue, nil),
			}
			return pdb, nil
		}
	}
}

// FrontLoadBalancerServiceReconciler returns the creator for the LoadBalancer that fronts apiserver
// when using exposeStrategy=LoadBalancer.
func FrontLoadBalancerServiceReconciler(data *resources.TemplateData) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return resources.FrontLoadBalancerServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			// We don't actually manage this service, that is done by the nodeport proxy, we just
			// must make sure that it exists
			s.Spec.Type = corev1.ServiceTypeLoadBalancer
			// Services need at least one port to be valid, so create it initially
			if len(s.Spec.Ports) == 0 {
				s.Spec.Ports = []corev1.ServicePort{
					{
						Name:       "secure",
						Port:       443,
						Protocol:   corev1.ProtocolTCP,
						TargetPort: intstr.FromInt(443),
					},
				}
			}

			seed := data.Seed()

			// seed.Spec.NodeportProxy.Annotations is deprecated and should be removed in the future
			// To avoid breaking changes we still copy these values over to the service annotations
			if seed.Spec.NodeportProxy.Annotations != nil {
				s.Annotations = seed.Spec.NodeportProxy.Annotations
			}

			if s.Annotations == nil {
				s.Annotations = make(map[string]string)
			}

			// Copy custom annotations specified for the loadBalancer Service. They have a higher precedence then
			// the common annotations specified in seed.Spec.NodeportProxy.Annotations, which is deprecated.
			if seed.Spec.NodeportProxy.Envoy.LoadBalancerService.Annotations != nil {
				for k, v := range seed.Spec.NodeportProxy.Envoy.LoadBalancerService.Annotations {
					s.Annotations[k] = v
				}
			}

			// set of Source IP ranges
			sourceIPList := sets.Set[string]{}

			if data.Seed().Spec.NodeportProxy.Envoy.LoadBalancerService.SourceRanges != nil {
				for _, cidr := range data.Seed().Spec.NodeportProxy.Envoy.LoadBalancerService.SourceRanges {
					sourceIPList.Insert(string(cidr))
				}
			}

			// Check if allowed IP ranges are configured and set the LoadBalancer source ranges
			if data.Cluster().Spec.APIServerAllowedIPRanges != nil {
				sourceIPList.Insert(data.Cluster().Spec.APIServerAllowedIPRanges.CIDRBlocks...)
				if seed := data.Seed(); seed != nil {
					if len(seed.Spec.DefaultAPIServerAllowedIPRanges) > 0 {
						sourceIPList.Insert(seed.Spec.DefaultAPIServerAllowedIPRanges...)
					}
				}
			}

			s.Spec.LoadBalancerSourceRanges = sets.List(sourceIPList)
			// for wider compatibility, we also set the source ranges via the service.beta.kubernetes.io annotation
			if sourceIPList.Len() > 0 {
				s.Annotations[loadBalancerSourceRangesAnnotationKey] = strings.Join(sets.List(sourceIPList), ",")
			} else {
				delete(s.Annotations, loadBalancerSourceRangesAnnotationKey)
			}

			if data.Cluster().Spec.Cloud.AWS != nil {
				// NOTE: While KKP uses in-tree CCM for AWS, we use annotations defined in
				// https://github.com/kubernetes/kubernetes/blob/v1.22.2/staging/src/k8s.io/legacy-cloud-providers/aws/aws.go

				// Make sure to use Network Load Balancer with fixed IPs instead of Classic Load Balancer
				s.Annotations["service.beta.kubernetes.io/aws-load-balancer-type"] = "nlb"
				// Extend the default idle timeout (60s), e.g. to not timeout "kubectl logs -f"
				s.Annotations["service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout"] = "3600"
				// Load-balance across nodes in all zones to ensure HA if nodes in a DNS-selected zone are not available
				s.Annotations["service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled"] = "true"
			}
			s.Spec.Selector = resources.BaseAppLabels(envoyAppLabelValue, nil)
			return s, nil
		}
	}
}
