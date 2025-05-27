//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

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

package seedconstraintsynchronizer_test

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	eeconstraintcontroller "k8c.io/kubermatic/v2/pkg/ee/constraint-controller"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	constraintName = "constraint"
	seedNamespace  = "kubermatic"
	kind           = "RequiredLabel"
)

func TestGetClustersForConstraint(t *testing.T) {
	workerSelector, _ := workerlabel.LabelSelector("")

	testCases := []struct {
		name                     string
		constraint               *kubermaticv1.Constraint
		clusters                 []ctrlruntimeclient.Object
		expectedUnwantedClusters sets.Set[string]
		expectedDesiredClusters  sets.Set[string]
	}{
		{
			name: "scenario 1: get clusters without filters",
			constraint: genConstraintWithSelector(kubermaticv1.ConstraintSelector{
				Providers:     nil,
				LabelSelector: metav1.LabelSelector{},
			}, seedNamespace),
			clusters: []ctrlruntimeclient.Object{
				genCluster("cluster1", nil, false),
				genCluster("cluster2", nil, false),
			},
			expectedDesiredClusters: sets.New("cluster1", "cluster2"),
		},
		{
			name: "scenario 2: filter clusters with labels",
			constraint: genConstraintWithSelector(kubermaticv1.ConstraintSelector{
				Providers: nil,
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"test": "value"},
				},
			}, seedNamespace),
			clusters: []ctrlruntimeclient.Object{
				genCluster("cluster1", nil, false),
				genCluster("cluster2", map[string]string{"test": "value"}, false),
			},
			expectedUnwantedClusters: sets.New("cluster1"),
			expectedDesiredClusters:  sets.New("cluster2"),
		},
		{
			name: "scenario 3: filter clusters with providers",
			constraint: genConstraintWithSelector(kubermaticv1.ConstraintSelector{
				Providers:     []string{"fake"},
				LabelSelector: metav1.LabelSelector{},
			}, seedNamespace),
			clusters: []ctrlruntimeclient.Object{
				genCluster("cluster1", nil, true),
				genCluster("cluster2", nil, false),
			},
			expectedUnwantedClusters: sets.New("cluster1"),
			expectedDesiredClusters:  sets.New("cluster2"),
		},
		{
			name: "scenario 4: filter clusters with providers and labels",
			constraint: genConstraintWithSelector(kubermaticv1.ConstraintSelector{
				Providers: []string{"fake"},
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"test": "value"},
				},
			}, seedNamespace),
			clusters: []ctrlruntimeclient.Object{
				genCluster("cluster1", nil, false),
				genCluster("cluster2", map[string]string{"test": "value"}, true),
				genCluster("cluster3", map[string]string{"test": "value"}, false),
			},
			expectedUnwantedClusters: sets.New("cluster1", "cluster2"),
			expectedDesiredClusters:  sets.New("cluster3"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			cli := fake.
				NewClientBuilder().
				WithObjects(tc.clusters...).
				Build()

			clusterList := &kubermaticv1.ClusterList{}
			if err := cli.List(ctx, clusterList, &ctrlruntimeclient.ListOptions{LabelSelector: workerSelector}); err != nil {
				t.Fatal(err)
			}

			desiredList, unwantedList, err := eeconstraintcontroller.FilterClustersForConstraint(ctx, cli, tc.constraint, clusterList)
			if err != nil {
				t.Fatal(err)
			}

			resultSetDesired := sets.New[string]()
			for _, cluster := range desiredList {
				resultSetDesired.Insert(cluster.Name)
			}

			resultSetUnwanted := sets.New[string]()
			for _, cluster := range unwantedList {
				resultSetUnwanted.Insert(cluster.Name)
			}

			if !resultSetDesired.Equal(tc.expectedDesiredClusters) {
				t.Fatalf("received clusters differ from expected:\n%v", diff.SetDiff(tc.expectedDesiredClusters, resultSetDesired))
			}

			if !resultSetUnwanted.Equal(tc.expectedUnwantedClusters) {
				t.Fatalf("received clusters differ from expected:\n%v", diff.SetDiff(tc.expectedUnwantedClusters, resultSetUnwanted))
			}
		})
	}
}

func genCluster(name string, labels map[string]string, bringYourOwnProvider bool) *kubermaticv1.Cluster {
	cluster := generator.GenDefaultCluster()

	cluster.Name = name
	cluster.Labels = labels

	if bringYourOwnProvider {
		cluster.Spec.Cloud.Fake = nil
		cluster.Spec.Cloud.BringYourOwn = &kubermaticv1.BringYourOwnCloudSpec{}
	}

	return cluster
}

func genConstraintWithSelector(selector kubermaticv1.ConstraintSelector, namespace string) *kubermaticv1.Constraint {
	ct := generator.GenConstraint(constraintName, namespace, kind)
	ct.Spec.Selector = selector
	return ct
}
