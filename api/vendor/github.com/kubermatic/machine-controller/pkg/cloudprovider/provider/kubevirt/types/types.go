/*
Copyright 2019 The Machine Controller Authors.

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

package types

import (
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
)

type RawConfig struct {
	Kubeconfig       providerconfigtypes.ConfigVarString `json:"kubeconfig,omitempty"`
	CPUs             providerconfigtypes.ConfigVarString `json:"cpus,omitempty"`
	Memory           providerconfigtypes.ConfigVarString `json:"memory,omitempty"`
	Namespace        providerconfigtypes.ConfigVarString `json:"namespace,omitempty"`
	SourceURL        providerconfigtypes.ConfigVarString `json:"sourceURL,omitempty"`
	PVCSize          providerconfigtypes.ConfigVarString `json:"pvcSize,omitempty"`
	StorageClassName providerconfigtypes.ConfigVarString `json:"storageClassName,omitempty"`
}
