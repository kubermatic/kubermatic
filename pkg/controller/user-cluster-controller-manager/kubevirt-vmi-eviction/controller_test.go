/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package kubevirtvmieviction

import (
	"context"
	"testing"

	kubevirtv1 "kubevirt.io/api/core/v1"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var testScheme = fake.NewScheme()

func init() {
	utilruntime.Must(kubevirtv1.AddToScheme(testScheme))
	utilruntime.Must(clusterv1alpha1.AddToScheme(testScheme))
}

const (
	clusterNamespace   = "cluster-xyz"
	evacuationNodeName = "node-a"
	machineToDelete    = "machine1"
	machineToNotDelete = "machine2"
)

func genVirtualMachineInstance(vmiName string, toEvict map[string]bool) *kubevirtv1.VirtualMachineInstance {
	status := kubevirtv1.VirtualMachineInstanceStatus{}
	if toEvict[vmiName] {
		status.EvacuationNodeName = evacuationNodeName
	}
	return &kubevirtv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmiName,
			Namespace: clusterNamespace,
		},
		Spec:   kubevirtv1.VirtualMachineInstanceSpec{},
		Status: status,
	}
}
func genMachine(machineName string) *clusterv1alpha1.Machine {
	return &clusterv1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineName,
			Namespace: machineNamespace,
		},
	}
}

var (
	vmisToEvict = map[string]bool{
		machineToDelete:    true,
		machineToNotDelete: false,
	}
)

func TestReconcile(t *testing.T) {
	testCases := map[string]struct {
		evacuationNodeName string
		expecDeleted       bool
		vmis               []ctrlruntimeclient.Object
		vmisToEvict        map[string]bool
		machines           []ctrlruntimeclient.Object
		clusterPaused      bool
	}{
		"EvacuationNodeName set, cluster not paused: machine deletion": {
			// 2 existing vmis and corresponding machines.
			// One of them having status.EvictionNodeName set.
			// This one should be deleted.
			machines: []ctrlruntimeclient.Object{
				genMachine(machineToDelete),
				genMachine(machineToNotDelete),
			},
			vmisToEvict: vmisToEvict,
			vmis: []ctrlruntimeclient.Object{
				genVirtualMachineInstance(machineToDelete, vmisToEvict),
				genVirtualMachineInstance(machineToNotDelete, vmisToEvict),
			},
		},
		"EvacuationNodeName set, cluster paused: machine not deleted": {
			// Cluster is paused, no deletion,
			machines: []ctrlruntimeclient.Object{
				genMachine(machineToDelete),
				genMachine(machineToNotDelete),
			},
			vmisToEvict: vmisToEvict,
			vmis: []ctrlruntimeclient.Object{
				genVirtualMachineInstance(machineToDelete, vmisToEvict),
				genVirtualMachineInstance(machineToNotDelete, vmisToEvict),
			},
			clusterPaused: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			r := reconciler{
				log: kubermaticlog.NewDefault().Sugar(),
				infraClient: fake.NewClientBuilder().WithScheme(testScheme).
					WithObjects(tc.vmis...).Build(),
				userClient: fake.NewClientBuilder().WithScheme(testScheme).
					WithObjects(tc.machines...).Build(),
				clusterIsPaused: func(context.Context) (bool, error) {
					return tc.clusterPaused, nil
				},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: clusterNamespace, Name: machineToDelete}}
			_, err := r.Reconcile(context.Background(), request)
			if err != nil {
				t.Fatalf("Got err %q from Reconcile(), expected no error", err)
			}

			machine := &clusterv1alpha1.Machine{}
			for _, o := range tc.vmis {
				vmi := o.(*kubevirtv1.VirtualMachineInstance)
				err = r.userClient.Get(context.Background(),
					types.NamespacedName{Name: vmi.Name, Namespace: machineNamespace},
					machine)

				if vmisToEvict[vmi.Name] && !tc.clusterPaused {
					if !apierrors.IsNotFound(err) {
						t.Errorf("Got err %q, expected MC to be deleted", err)
					}
				} else {
					if apierrors.IsNotFound(err) {
						t.Errorf("This machine %s should not be deleted", machineToNotDelete)
					}
				}
			}
		})
	}
}
