package scheduler

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defaultMemoryRequest = resource.MustParse("64Mi")
	defaultCPURequest    = resource.MustParse("20m")
	defaultMemoryLimit   = resource.MustParse("128Mi")
	defaultCPULimit      = resource.MustParse("100m")
)

const (
	name = "scheduler"
)

// Deployment returns the kubernetes Controller-Manager Deployment
func Deployment(data *resources.TemplateData, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.SchedulerDeploymentName
	dep.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	dep.Labels = resources.GetLabels(name)

	dep.Spec.Replicas = resources.Int32(1)
	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			resources.AppLabelKey: name,
		},
	}
	dep.Spec.Strategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
	dep.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
		MaxSurge: &intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 1,
		},
		MaxUnavailable: &intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 0,
		},
	}

	dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Labels: map[string]string{
			resources.AppLabelKey: name,
		},
		Annotations: map[string]string{
			"prometheus.io/scrape": "true",
			"prometheus.io/path":   "/metrics",
			"prometheus.io/port":   "10251",
		},
	}

	// get clusterIP of apiserver
	apiserverService, err := data.ServiceLister.Services(data.Cluster.Status.NamespaceName).Get("apiserver")
	if err != nil {
		return nil, fmt.Errorf("apiserver service in namespace %s not found: %v",
			data.Cluster.Status.NamespaceName, err)
	}

	dep.Spec.Template.Spec.InitContainers = []corev1.Container{
		{
			Name:            "apiserver-running",
			Image:           data.ImageRegistry("docker.io") + "/busybox",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command: []string{
				"/bin/sh",
				"-ec",
				"until wget -T 1 http://" + apiserverService.Spec.ClusterIP + ":8080/healthz; do echo waiting for apiserver; sleep 2; done;",
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		},
	}
	dep.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:            name,
			Image:           data.ImageRegistry("gcr.io") + "/google_containers/hyperkube-amd64:v" + data.Cluster.Spec.Version,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/hyperkube", "scheduler"},
			Args: []string{
				"--master", fmt.Sprintf("http://%s:8080", apiserverService.Spec.ClusterIP),
				"--v", "4",
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: defaultMemoryRequest,
					corev1.ResourceCPU:    defaultCPURequest,
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: defaultMemoryLimit,
					corev1.ResourceCPU:    defaultCPULimit,
				},
			},
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt(10251),
					},
				},
				FailureThreshold: 3,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				TimeoutSeconds:   15,
			},
			LivenessProbe: &corev1.Probe{
				FailureThreshold: 8,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt(10251),
					},
				},
				InitialDelaySeconds: 15,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				TimeoutSeconds:      15,
			},
		},
	}

	return dep, nil
}
