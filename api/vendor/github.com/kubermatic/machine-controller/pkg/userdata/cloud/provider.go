package cloud

import (
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type ConfigProvider interface {
	GetCloudConfig(spec clusterv1alpha1.MachineSpec) (config string, name string, err error)
}
