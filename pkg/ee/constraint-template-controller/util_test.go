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

package constrainttemplatecontroller_test

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	eectcontroller "k8c.io/kubermatic/v2/pkg/ee/constraint-template-controller"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetClustersForConstraintTemplate(t *testing.T) {
	workerSelector, _ := workerlabel.LabelSelector("")

	testCases := []struct {
		name             string
		ct               *kubermaticv1.ConstraintTemplate
		clusters         []ctrlruntimeclient.Object
		expectedClusters sets.Set[string]
	}{
		{
			name: "scenario 1: get clusters without filters",
			ct: genConstraintTemplateWithSelector(kubermaticv1.ConstraintTemplateSelector{
				Providers:     nil,
				LabelSelector: metav1.LabelSelector{},
			}),
			clusters: []ctrlruntimeclient.Object{
				genCluster("cluster1", nil, false),
				genCluster("cluster2", nil, false),
			},
			expectedClusters: sets.New("cluster1", "cluster2"),
		},
		{
			name: "scenario 2: filter clusters with labels",
			ct: genConstraintTemplateWithSelector(kubermaticv1.ConstraintTemplateSelector{
				Providers: nil,
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"test": "value"},
				},
			}),
			clusters: []ctrlruntimeclient.Object{
				genCluster("cluster1", map[string]string{"test": "value"}, false),
				genCluster("cluster2", nil, false),
			},
			expectedClusters: sets.New("cluster1"),
		},
		{
			name: "scenario 3: filter clusters with providers",
			ct: genConstraintTemplateWithSelector(kubermaticv1.ConstraintTemplateSelector{
				Providers:     []string{"fake"},
				LabelSelector: metav1.LabelSelector{},
			}),
			clusters: []ctrlruntimeclient.Object{
				genCluster("cluster1", nil, false),
				genCluster("cluster2", nil, true),
			},
			expectedClusters: sets.New("cluster1"),
		},
		{
			name: "scenario 4: filter clusters with providers and labels",
			ct: genConstraintTemplateWithSelector(kubermaticv1.ConstraintTemplateSelector{
				Providers: []string{"fake"},
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"test": "value"},
				},
			}),
			clusters: []ctrlruntimeclient.Object{
				genCluster("cluster1", nil, false),
				genCluster("cluster2", map[string]string{"test": "value"}, true),
				genCluster("cluster3", map[string]string{"test": "value"}, false),
			},
			expectedClusters: sets.New("cluster3"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := fake.
				NewClientBuilder().
				WithObjects(tc.clusters...).
				Build()

			clusterList, err := eectcontroller.GetClustersForConstraintTemplate(context.Background(), cli, tc.ct, workerSelector)
			if err != nil {
				t.Fatal(err)
			}

			resultSet := sets.New[string]()
			for _, cluster := range clusterList.Items {
				resultSet.Insert(cluster.Name)
			}

			if !resultSet.Equal(tc.expectedClusters) {
				t.Fatalf("received clusters differ from expected:\n%v", diff.SetDiff(tc.expectedClusters, resultSet))
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

func genConstraintTemplateWithSelector(selector kubermaticv1.ConstraintTemplateSelector) *kubermaticv1.ConstraintTemplate {
	ct := generator.GenConstraintTemplate("ct1")
	ct.Spec.Selector = selector
	return ct
}
