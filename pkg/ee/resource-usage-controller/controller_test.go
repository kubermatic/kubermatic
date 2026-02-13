//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package resourceusagecontroller

import (
	"context"
	"fmt"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                  string
		cluster               *kubermaticv1.Cluster
		machines              []*clusterv1alpha1.Machine
		expectedResourceUsage *kubermaticv1.ResourceDetails
	}{
		{
			name:     "scenario 1: calculate resource usage from one machine",
			cluster:  generator.GenDefaultCluster(),
			machines: []*clusterv1alpha1.Machine{genFakeMachine("m1", "5", "5G", "10G")},
			expectedResourceUsage: &kubermaticv1.ResourceDetails{
				CPU:     getQuantity("5"),
				Memory:  getQuantity("5G"),
				Storage: getQuantity("10G"),
			},
		},
		{
			name: "scenario 2: set proper resource usage",
			cluster: func() *kubermaticv1.Cluster {
				c := generator.GenDefaultCluster()
				c.Status.ResourceUsage = kubermaticv1.NewResourceDetails(resource.MustParse("2"), resource.MustParse("1G"), resource.MustParse("2G"))
				return c
			}(),
			machines: []*clusterv1alpha1.Machine{genFakeMachine("m1", "5", "5G", "10G")},
			expectedResourceUsage: &kubermaticv1.ResourceDetails{
				CPU:     getQuantity("5"),
				Memory:  getQuantity("5G"),
				Storage: getQuantity("10G"),
			},
		},
		{
			name:    "scenario 3: calculate proper resource usage from 2 machines",
			cluster: generator.GenDefaultCluster(),
			machines: []*clusterv1alpha1.Machine{
				genFakeMachine("m1", "5", "5G", "10G"),
				genFakeMachine("m2", "2", "3G", "5G")},
			expectedResourceUsage: &kubermaticv1.ResourceDetails{
				CPU:     getQuantity("7"),
				Memory:  getQuantity("8G"),
				Storage: getQuantity("15G"),
			},
		},
		{
			name: "scenario 4: set zero usage if no machines",
			cluster: func() *kubermaticv1.Cluster {
				c := generator.GenDefaultCluster()
				c.Status.ResourceUsage = kubermaticv1.NewResourceDetails(resource.MustParse("2"), resource.MustParse("1G"), resource.MustParse("2G"))
				return c
			}(),
			expectedResourceUsage: &kubermaticv1.ResourceDetails{
				CPU:     getQuantity("0"),
				Memory:  getQuantity("0"),
				Storage: getQuantity("0"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			scheme := fake.NewScheme()
			utilruntime.Must(clusterv1alpha1.AddToScheme(scheme))

			seedClientBuilder := fake.NewClientBuilder().WithScheme(scheme)
			seedClientBuilder.WithObjects(tc.cluster)

			userClientBuilder := fake.NewClientBuilder().WithScheme(scheme)
			for _, m := range tc.machines {
				userClientBuilder.WithObjects(m)
			}

			seedClient := seedClientBuilder.Build()
			userClient := userClientBuilder.Build()

			r := reconciler{
				log:         kubermaticlog.Logger,
				seedClient:  seedClient,
				userClient:  userClient,
				clusterName: tc.cluster.Name,
				caBundle:    nil,
				recorder:    &events.FakeRecorder{},
				clusterIsPaused: func(c context.Context) (bool, error) {
					return false, nil
				},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.cluster.Name}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			// check Cluster
			cluster := &kubermaticv1.Cluster{}
			err := seedClient.Get(ctx, types.NamespacedName{Name: tc.cluster.Name}, cluster)
			if err != nil {
				t.Fatalf("failed to get cluster: %v", err)
			}

			if !diff.SemanticallyEqual(tc.expectedResourceUsage, cluster.Status.ResourceUsage) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedResourceUsage, cluster.Status.ResourceUsage))
			}
		})
	}
}

func genFakeMachine(name, cpu, memory, storage string) *clusterv1alpha1.Machine {
	return generator.GenTestMachine(name,
		fmt.Sprintf(`{"cloudProvider":"fake", "cloudProviderSpec":{"cpu":"%s","memory":"%s","storage":"%s"}}`, cpu, memory, storage),
		nil, nil)
}

func getQuantity(q string) *resource.Quantity {
	res := resource.MustParse(q)
	return &res
}
