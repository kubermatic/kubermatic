package cloud

import (
	machinesv1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
)

type ConfigProvider interface {
	GetCloudConfig(spec machinesv1alpha1.MachineSpec) (config string, name string, err error)
}
