package apiserver

import (
	"encoding/json"
	"errors"
	"fmt"

	httpproberapi "github.com/kubermatic/kubermatic/api/cmd/http-prober/api"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	tag                = "v0.3.1"
	emptyDirVolumeName = "http-prober-bin"
	initContainerName  = "copy-http-prober"
)

// IsRunningInitContainer returns a init container which will wait until the apiserver is reachable via its ClusterIP
type isRunningInitContainerData interface {
	ImageRegistry(string) string
	Cluster() *kubermaticv1.Cluster
}

// IsRunningWrapper wraps the named containers in the pod with a check if the API server is reachable.
// This is achieved by copying a `http-prober` binary via an init container into an emptyDir volume,
// then mounting that volume onto all named containers and replacing the command with a call to
// the `http-prober` binary. The http prober binary gets the original command as serialized string
// and does an syscall.Exec onto it once the apiserver became reachable
func IsRunningWrapper(data isRunningInitContainerData, spec corev1.PodSpec, containersToWrap sets.String, crdsToWaitFor ...string) (*corev1.PodSpec, error) {
	if containersToWrap.Len() == 0 {
		return nil, errors.New("no containers to wrap passed")
	}

	for _, containerToWrap := range containersToWrap.List() {
		if !hasContainerNamed(spec, containerToWrap) {
			return nil, fmt.Errorf("pod has no container named %q", containerToWrap)
		}
	}

	var newVolumes []corev1.Volume
	for _, volume := range spec.Volumes {
		if volume.Name == emptyDirVolumeName {
			continue
		}
		newVolumes = append(newVolumes, volume)
	}
	newVolumes = append(newVolumes, corev1.Volume{
		Name: emptyDirVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	spec.Volumes = newVolumes

	var newInitContainers []corev1.Container
	for _, container := range spec.InitContainers {
		if container.Name == initContainerName {
			continue
		}
		newInitContainers = append(newInitContainers, container)
	}
	copyContainer := corev1.Container{
		Name:    initContainerName,
		Image:   data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/http-prober:" + tag,
		Command: []string{"/bin/cp", "/usr/local/bin/http-prober", "/http-prober-bin/http-prober"},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      emptyDirVolumeName,
			MountPath: "/http-prober-bin",
		}},
	}
	// We must come first in case an initContainer gets wrapped
	spec.InitContainers = append([]corev1.Container{copyContainer}, newInitContainers...)

	for idx := range spec.InitContainers {
		if !containersToWrap.Has(spec.InitContainers[idx].Name) {
			continue
		}
		wrappedContainer, err := wrapContainer(data, spec.InitContainers[idx], crdsToWaitFor...)
		if err != nil {
			return nil, fmt.Errorf("failed to wrap initContainer %q: %v", spec.InitContainers[idx].Name, err)
		}
		spec.InitContainers[idx] = *wrappedContainer
	}
	for idx := range spec.Containers {
		if !containersToWrap.Has(spec.Containers[idx].Name) {
			continue
		}
		wrappedContainer, err := wrapContainer(data, spec.Containers[idx], crdsToWaitFor...)
		if err != nil {
			return nil, fmt.Errorf("failed to wrap container %q: %v", spec.Containers[idx].Name, err)
		}
		spec.Containers[idx] = *wrappedContainer
	}

	return &spec, nil
}

func hasContainerNamed(spec corev1.PodSpec, name string) bool {
	for _, container := range append(spec.InitContainers, spec.Containers...) {
		if container.Name == name {
			return true
		}
	}
	return false
}

func wrapContainer(data isRunningInitContainerData, container corev1.Container, crdsToWaitFor ...string) (*corev1.Container, error) {
	commandWithArgs := append(container.Command, container.Args...)
	if len(commandWithArgs) == 0 {
		return nil, fmt.Errorf("container %q has no command or args set", container.Name)
	}
	command := httpproberapi.Command{
		Command: commandWithArgs[0],
	}
	if len(commandWithArgs) > 1 {
		command.Args = commandWithArgs[1:]
	}
	serializedCommand, err := json.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %v", err)
	}

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      emptyDirVolumeName,
		MountPath: "/http-prober-bin",
	})
	container.Command = []string{"/http-prober-bin/http-prober"}
	container.Args = []string{
		"-endpoint", fmt.Sprintf("https://%s:%d/healthz", data.Cluster().Address.InternalName, data.Cluster().Address.Port),
		"-insecure",
		"-retries", "100",
		"-retry-wait", "2",
		"-timeout", "1",
		"-command", string(serializedCommand),
	}
	for _, crdToWaitFor := range crdsToWaitFor {
		container.Args = append(container.Args, "--crd-to-wait-for", crdToWaitFor)
	}

	return &container, nil
}
