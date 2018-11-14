package cloud

import (
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"

	"k8s.io/apimachinery/pkg/types"
	listerscorev1 "k8s.io/client-go/listers/core/v1"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// Provider exposed all required functions to interact with a cloud provider
type Provider interface {
	// AddDefaults will read the MachineSpec and apply defaults for provider specific fields
	AddDefaults(spec clusterv1alpha1.MachineSpec) (clusterv1alpha1.MachineSpec, error)

	// Validate validates the given machine's specification.
	//
	// In case of any error a "terminal" error should be set,
	// See v1alpha1.MachineStatus for more info
	Validate(machinespec clusterv1alpha1.MachineSpec) error

	// Get gets a node that is associated with the given machine.
	//
	// Note that this method can return what we call a "terminal" error,
	// which indicates that a manual interaction is required to recover from this state.
	// See v1alpha1.MachineStatus for more info and TerminalError type
	//
	// In case the instance cannot be found, github.com/kubermatic/machine-controller/pkg/cloudprovider/errors/ErrInstanceNotFound will be returned
	Get(machine *clusterv1alpha1.Machine) (instance.Instance, error)

	// GetCloudConfig will return the cloud provider specific cloud-config, which gets consumed by the kubelet
	GetCloudConfig(spec clusterv1alpha1.MachineSpec) (config string, name string, err error)

	// Create creates a cloud instance according to the given machine
	Create(machine *clusterv1alpha1.Machine, data *MachineCreateDeleteData, userdata string) (instance.Instance, error)

	// Delete deletes the instance and all associated ressources
	// This will always be called on machine deletion, the implemention must check if there is actually
	// something to delete and just do nothing if there isn't
	// In case the instance is already gone, nil will be returned
	Delete(machine *clusterv1alpha1.Machine, data *MachineCreateDeleteData) error

	// MachineMetricsLabels returns labels used for the Prometheus metrics
	// about created machines, e.g. instance type, instance size, region
	// or whatever the provider deems interesting. Should always return
	// a "size" label.
	MachineMetricsLabels(machine *clusterv1alpha1.Machine) (map[string]string, error)

	// MigrateUID is called when the controller migrates types and the UID of the machine object changes
	// All cloud providers that use Machine.UID to uniquely identify resources must implement this
	MigrateUID(machine *clusterv1alpha1.Machine, new types.UID) error
}

// MachineUpdater defines a function to persist an update to a machine
type MachineUpdater func(*clusterv1alpha1.Machine, func(*clusterv1alpha1.Machine)) (*clusterv1alpha1.Machine, error)

// MachineCreateDeleteData is the struct the cloud providers get when creating or deleting an instance
type MachineCreateDeleteData struct {
	Updater  MachineUpdater
	PVLister listerscorev1.PersistentVolumeLister
}
