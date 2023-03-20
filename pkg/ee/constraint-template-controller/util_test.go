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

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	eectcontroller "k8c.io/kubermatic/v2/pkg/ee/constraint-template-controller"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
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
				genCluster("cluster1", nil, kubermaticv1.CloudProviderAWS),
				genCluster("cluster2", nil, kubermaticv1.CloudProviderAWS),
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
				genCluster("cluster1", map[string]string{"test": "value"}, kubermaticv1.CloudProviderAWS),
				genCluster("cluster2", nil, kubermaticv1.CloudProviderAWS),
			},
			expectedClusters: sets.New("cluster1"),
		},
		{
			name: "scenario 3: filter clusters with providers",
			ct: genConstraintTemplateWithSelector(kubermaticv1.ConstraintTemplateSelector{
				Providers:     []kubermaticv1.CloudProvider{kubermaticv1.CloudProviderAWS},
				LabelSelector: metav1.LabelSelector{},
			}),
			clusters: []ctrlruntimeclient.Object{
				genCluster("cluster1", nil, kubermaticv1.CloudProviderAWS),
				genCluster("cluster2", nil, kubermaticv1.CloudProviderBringYourOwn),
			},
			expectedClusters: sets.New("cluster1"),
		},
		{
			name: "scenario 4: filter clusters with providers and labels",
			ct: genConstraintTemplateWithSelector(kubermaticv1.ConstraintTemplateSelector{
				Providers: []kubermaticv1.CloudProvider{kubermaticv1.CloudProviderAWS},
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"test": "value"},
				},
			}),
			clusters: []ctrlruntimeclient.Object{
				genCluster("cluster1", nil, kubermaticv1.CloudProviderAWS),
				genCluster("cluster2", map[string]string{"test": "value"}, kubermaticv1.CloudProviderBringYourOwn),
				genCluster("cluster3", map[string]string{"test": "value"}, kubermaticv1.CloudProviderAWS),
			},
			expectedClusters: sets.New("cluster3"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
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

func genCluster(name string, labels map[string]string, cloudProvider kubermaticv1.CloudProvider) *kubermaticv1.Cluster {
	cluster := generator.GenDefaultCluster()

	cluster.Name = name
	cluster.Labels = labels
	cluster.Spec.Cloud.ProviderName = cloudProvider
	cluster.Spec.Cloud.BringYourOwn = nil

	switch cloudProvider {
	case kubermaticv1.CloudProviderBringYourOwn:
		cluster.Spec.Cloud.BringYourOwn = &kubermaticv1.BringYourOwnCloudSpec{}
	case kubermaticv1.CloudProviderAWS:
		cluster.Spec.Cloud.AWS = &kubermaticv1.AWSCloudSpec{}
	default:
		panic("Only AWS and BYO are implemented for this test.")
	}

	return cluster
}

func genConstraintTemplateWithSelector(selector kubermaticv1.ConstraintTemplateSelector) *kubermaticv1.ConstraintTemplate {
	ct := generator.GenConstraintTemplate("ct1")
	ct.Spec.Selector = selector
	return ct
}
