/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package provider

import (
	"fmt"
	"reflect"
	"testing"

	kvinstancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_filterInstancetypes(t *testing.T) {
	tests := []struct {
		name          string
		instancetypes *apiv2.VirtualMachineInstancetypeList
		quota         kubermaticv1.MachineDeploymentVMResourceQuota
		want          *apiv2.VirtualMachineInstancetypeList
	}{
		{
			name: "some filtered out",
			instancetypes: newInstancetypeList().
				addInstanceType(apiv2.InstancetypeCustom, 2, "4Gi").     // ok
				addInstanceType(apiv2.InstancetypeKubermatic, 3, "4Gi"). // ok
				addInstanceType(apiv2.InstancetypeCustom, 4, "4Gi").     // ok
				addInstanceType(apiv2.InstancetypeKubermatic, 4, "4Gi"). // ok
				addInstanceType(apiv2.InstancetypeCustom, 1, "4Gi").     // filtered out due to cpu
				addInstanceType(apiv2.InstancetypeCustom, 5, "4Gi").     // filtered out due to cpu
				addInstanceType(apiv2.InstancetypeCustom, 2, "2Gi").     // filtered out due to memory
				addInstanceType(apiv2.InstancetypeCustom, 2, "6Gi").     // filtered out due to memory
				toApiWithoutError(),
			quota: kubermaticv1.MachineDeploymentVMResourceQuota{
				MinCPU: 2,
				MaxCPU: 4,
				MinRAM: 3,
				MaxRAM: 5,
			},
			want: newInstancetypeList().
				addInstanceType(apiv2.InstancetypeCustom, 2, "4Gi").     // ok
				addInstanceType(apiv2.InstancetypeKubermatic, 3, "4Gi"). // ok
				addInstanceType(apiv2.InstancetypeCustom, 4, "4Gi").     // ok
				addInstanceType(apiv2.InstancetypeKubermatic, 4, "4Gi"). // ok
				toApiWithoutError(),
		},
		{
			name: "some filtered out-due to units",
			instancetypes: newInstancetypeList().
				addInstanceType(apiv2.InstancetypeCustom, 2, "4Mi").     // filtered out due to memory
				addInstanceType(apiv2.InstancetypeKubermatic, 3, "4Ti"). // filtered out due to memory
				toApiWithoutError(),
			quota: kubermaticv1.MachineDeploymentVMResourceQuota{
				MinCPU: 2,
				MaxCPU: 4,
				MinRAM: 3,
				MaxRAM: 5,
			},
			want: newInstancetypeList().
				toApiWithoutError(),
		},
		{
			name: "all filtered out",
			instancetypes: newInstancetypeList().
				addInstanceType(apiv2.InstancetypeCustom, 1, "4Gi").     // filtered out due to cpu
				addInstanceType(apiv2.InstancetypeKubermatic, 5, "4Gi"). // filtered out due to cpu
				addInstanceType(apiv2.InstancetypeCustom, 2, "2Gi").     // filtered out due to memory
				addInstanceType(apiv2.InstancetypeKubermatic, 2, "6Gi"). // filtered out due to memory
				toApiWithoutError(),
			quota: kubermaticv1.MachineDeploymentVMResourceQuota{
				MinCPU: 2,
				MaxCPU: 4,
				MinRAM: 3,
				MaxRAM: 5,
			},
			want: newInstancetypeList().
				toApiWithoutError(),
		},
		{
			name: "all custom-none filtered out",
			instancetypes: newInstancetypeList().
				addInstanceType(apiv2.InstancetypeCustom, 2, "4Gi").     // ok
				addInstanceType(apiv2.InstancetypeKubermatic, 3, "4Gi"). // ok
				addInstanceType(apiv2.InstancetypeCustom, 4, "4Gi").     // ok
				addInstanceType(apiv2.InstancetypeKubermatic, 4, "4Gi"). // ok
				toApiWithoutError(),
			quota: kubermaticv1.MachineDeploymentVMResourceQuota{
				MinCPU: 2,
				MaxCPU: 4,
				MinRAM: 3,
				MaxRAM: 5,
			},
			want: newInstancetypeList().
				addInstanceType(apiv2.InstancetypeCustom, 2, "4Gi").     // ok
				addInstanceType(apiv2.InstancetypeKubermatic, 3, "4Gi"). // ok
				addInstanceType(apiv2.InstancetypeCustom, 4, "4Gi").     // ok
				addInstanceType(apiv2.InstancetypeKubermatic, 4, "4Gi"). // ok
				toApiWithoutError(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterInstancetypes(tt.instancetypes, tt.quota); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got\n %+v", got.Instancetypes == nil)
				t.Errorf("want\n %+v", tt.want.Instancetypes == nil)

				t.Errorf("filterInstancetypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newInstancetypeList() *instancetypeListWrapper {
	return &instancetypeListWrapper{}
}

func (l *instancetypeListWrapper) toApiWithoutError() *apiv2.VirtualMachineInstancetypeList {
	res, err := l.toApi()
	if err != nil {
		return nil
	}
	return res
}

func (l *instancetypeListWrapper) addInstanceType(category apiv2.VirtualMachineInstancetypeCategory, cpu uint32, memory string) *instancetypeListWrapper {
	w := newInstancetype(category, cpu, memory)

	if l.items == nil {
		l.items = make([]instancetypeWrapper, 0)
	}
	l.items = append(l.items, w)
	return l
}

func newInstancetype(category apiv2.VirtualMachineInstancetypeCategory, cpu uint32, memory string) instancetypeWrapper {
	if category == apiv2.InstancetypeKubermatic {
		instancetype := &kvinstancetypev1alpha1.VirtualMachineInstancetype{
			ObjectMeta: metav1.ObjectMeta{
				Name: instancetypeName(cpu, memory),
			},
			Spec: getInstancetypeSpec(cpu, memory),
		}

		return &standardInstancetypeWrapper{instancetype}
	} else if category == apiv2.InstancetypeCustom {
		instancetype := &kvinstancetypev1alpha1.VirtualMachineClusterInstancetype{
			ObjectMeta: metav1.ObjectMeta{
				Name: "small-1",
			},
			Spec: getInstancetypeSpec(cpu, memory),
		}
		return &customInstancetypeWrapper{instancetype}
	}
	return nil
}

func instancetypeName(cpu uint32, memory string) string {
	return fmt.Sprintf("cpu-%d-memory-%s", cpu, memory)
}

func getQuantity(q string) *resource.Quantity {
	res := resource.MustParse(q)
	return &res
}

func getInstancetypeSpec(cpu uint32, memory string) kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec {
	return kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{
		CPU: kvinstancetypev1alpha1.CPUInstancetype{
			Guest: cpu,
		},
		Memory: kvinstancetypev1alpha1.MemoryInstancetype{
			Guest: *getQuantity(memory),
		},
	}
}
