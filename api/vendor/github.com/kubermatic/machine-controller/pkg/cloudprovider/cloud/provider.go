package cloud

import (
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
)

// Provider exposed all required functions to interact with a cloud provider
type Provider interface {
	AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, bool, error)

	// Validate validates the given machine's specification.
	//
	// In case of any error a "terminal" error should be set,
	// See v1alpha1.MachineStatus for more info
	Validate(machinespec v1alpha1.MachineSpec) error

	// Get gets a node that is associated with the given machine.
	//
	// Note that this method can return what we call a "terminal" error,
	// which indicates that a manual interaction is required to recover from this state.
	// See v1alpha1.MachineStatus for more info and TerminalError type
	Get(machine *v1alpha1.Machine) (instance.Instance, error)

	GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error)

	// Create creates a cloud instance according to the given machine
	Create(machine *v1alpha1.Machine, update MachineUpdater, userdata string) (instance.Instance, error)

	// Delete deletes the instance and all associated ressources
	// This will always be called on machine deletion, the implemention must check if there is actually
	// something to delete and just do nothing if there isn't
	Delete(machine *v1alpha1.Machine, update MachineUpdater) error

	// MachineMetricsLabels returns labels used for the Prometheus metrics
	// about created machines, e.g. instance type, instance size, region
	// or whatever the provider deems interesting. Should always return
	// a "size" label.
	MachineMetricsLabels(machine *v1alpha1.Machine) (map[string]string, error)
}

// MachineUpdater defines a function to persist an update to a machine
type MachineUpdater func(string, func(*v1alpha1.Machine)) (*v1alpha1.Machine, error)
