package machinecontroller

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	name = "machine-controller"

	tag = "v0.7.3"
)

// Deployment returns the machine-controller Deployment
func Deployment(data *resources.TemplateData, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.MachineControllerDeploymentName
	dep.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	dep.Labels = resources.GetLabels(name)

	dep.Spec.Replicas = resources.Int32(1)
	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			resources.AppLabelKey: "machine-controller",
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
			"prometheus.io/port":   "8085",
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
			Image:           data.ImageRegistry("docker.io") + "/kubermatic/machine-controller:" + tag,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/usr/local/bin/machine-controller"},
			Args: []string{
				"-master", fmt.Sprintf("http://%s:8080", apiserverService.Spec.ClusterIP),
				"-logtostderr",
				"-v", "4",
				"-cluster-dns", "10.10.10.10",
				"-internal-listen-address", "0.0.0.0:8085",
			},
			Env: getEnvVars(data),
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/ready",
						Port: intstr.FromInt(8085),
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
						Path: "/live",
						Port: intstr.FromInt(8085),
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

func getEnvVars(data *resources.TemplateData) []corev1.EnvVar {
	var vars []corev1.EnvVar
	if data.Cluster.Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", Value: data.Cluster.Spec.Cloud.AWS.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", Value: data.Cluster.Spec.Cloud.AWS.SecretAccessKey})
	}
	if data.Cluster.Spec.Cloud.Openstack != nil {
		vars = append(vars, corev1.EnvVar{Name: "OS_AUTH_URL", Value: data.DC.Spec.Openstack.AuthURL})
		vars = append(vars, corev1.EnvVar{Name: "OS_USER_NAME", Value: data.Cluster.Spec.Cloud.Openstack.Username})
		vars = append(vars, corev1.EnvVar{Name: "OS_PASSWORD", Value: data.Cluster.Spec.Cloud.Openstack.Password})
		vars = append(vars, corev1.EnvVar{Name: "OS_DOMAIN_NAME", Value: data.Cluster.Spec.Cloud.Openstack.Domain})
		vars = append(vars, corev1.EnvVar{Name: "OS_TENANT_NAME", Value: data.Cluster.Spec.Cloud.Openstack.Tenant})
	}
	if data.Cluster.Spec.Cloud.Hetzner != nil {
		vars = append(vars, corev1.EnvVar{Name: "HZ_TOKEN", Value: data.Cluster.Spec.Cloud.Hetzner.Token})
	}
	if data.Cluster.Spec.Cloud.Digitalocean != nil {
		vars = append(vars, corev1.EnvVar{Name: "DO_TOKEN", Value: data.Cluster.Spec.Cloud.Digitalocean.Token})
	}
	return vars
}
