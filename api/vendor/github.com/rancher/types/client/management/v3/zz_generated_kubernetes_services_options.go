package client

const (
	KubernetesServicesOptionsType                = "kubernetesServicesOptions"
	KubernetesServicesOptionsFieldKubeAPI        = "kubeapi"
	KubernetesServicesOptionsFieldKubeController = "kubeController"
	KubernetesServicesOptionsFieldKubelet        = "kubelet"
	KubernetesServicesOptionsFieldKubeproxy      = "kubeproxy"
	KubernetesServicesOptionsFieldScheduler      = "scheduler"
)

type KubernetesServicesOptions struct {
	KubeAPI        map[string]string `json:"kubeapi,omitempty" yaml:"kubeapi,omitempty"`
	KubeController map[string]string `json:"kubeController,omitempty" yaml:"kubeController,omitempty"`
	Kubelet        map[string]string `json:"kubelet,omitempty" yaml:"kubelet,omitempty"`
	Kubeproxy      map[string]string `json:"kubeproxy,omitempty" yaml:"kubeproxy,omitempty"`
	Scheduler      map[string]string `json:"scheduler,omitempty" yaml:"scheduler,omitempty"`
}
