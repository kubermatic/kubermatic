package kubevirtvmievictioncontroller

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func genVirtualMachineInstance(vmiName, evacuationNodeName string) *kubevirtv1.VirtualMachineInstance {
	return &kubevirtv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmiName,
			Namespace: fmt.Sprintf("cluster-%s", vmiName),
		},
		Spec: kubevirtv1.VirtualMachineInstanceSpec{},
		Status: kubevirtv1.VirtualMachineInstanceStatus{
			EvacuationNodeName: evacuationNodeName,
		},
	}
}
func genMachine(clusterName string) *v1alpha1.Machine {
	return &v1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: machineNamespace,
		},
	}
}

func TestReconcile(t *testing.T) {

	scheme := runtime.NewScheme()
	_ = kubevirtv1.AddToScheme(scheme)
	_ = clusterv1alpha1.AddToScheme(scheme)

	testCases := map[string]struct {
		vmiName            string
		evacuationNodeName string
		expecDeleted       bool
	}{
		"No EvacuationNodeName: No Machine deletion": {
			vmiName:      "test-vmi",
			expecDeleted: false,
		},
		"EvacuationNodeName set: Machine deletion": {
			vmiName:            "test-vmi",
			evacuationNodeName: "nodeA",
			expecDeleted:       true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			r := reconciler{
				log:         kubermaticlog.NewDefault().Sugar(),
				infraClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(genVirtualMachineInstance(tc.vmiName, tc.evacuationNodeName)).Build(),
				userClient:  fake.NewClientBuilder().WithScheme(scheme).WithObjects(genMachine(tc.vmiName)).Build(),
				clusterIsPaused: func(context.Context) (bool, error) {
					return false, nil
				},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: fmt.Sprintf("cluster-%s", tc.vmiName), Name: tc.vmiName}}
			_, err := r.Reconcile(context.Background(), request)
			if err != nil {
				t.Fatalf("Got err %q from Reconcile(), expected no error", err)
			}

			namepacedMachineName := types.NamespacedName{Name: tc.vmiName, Namespace: machineNamespace}
			machine := &v1alpha1.Machine{}
			err = r.userClient.Get(context.Background(), namepacedMachineName, machine)
			if tc.expecDeleted && !errors.IsNotFound(err) {
				t.Errorf("Got err %q, expected MC to be deleted", err)
			}

		})

	}
}
