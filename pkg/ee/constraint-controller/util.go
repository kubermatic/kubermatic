// +build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Loodse GmbH

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

package seedconstraintsynchronizer

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetClustersForConstraint gets clusters for the constraints by using the constraints selector to filter out unselected clusters
func GetClustersForConstraint(ctx context.Context, client ctrlruntimeclient.Client,
	constraint *kubermaticv1.Constraint, workerNamesLabelSelector labels.Selector) (*kubermaticv1.ClusterList, error) {

	constraintLabelSelector, err := v1.LabelSelectorAsSelector(&constraint.Spec.Selector.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("error converting Constraint label selector (%v) to a kubernetes selector: %w", constraint.Spec.Selector.LabelSelector, err)
	}

	var selector labels.Selector
	constraintReq, _ := constraintLabelSelector.Requirements()
	selector = workerNamesLabelSelector.Add(constraintReq...)

	clusterList := &kubermaticv1.ClusterList{}

	if err := client.List(ctx, clusterList, &ctrlruntimeclient.ListOptions{LabelSelector: selector}); err != nil {
		return nil, fmt.Errorf("failed listing clusters: %w", err)
	}

	clusterList.Items = filterProvider(constraint, clusterList.Items)
	return clusterList, nil
}

func filterProvider(constraint *kubermaticv1.Constraint, ctList []kubermaticv1.Cluster) []kubermaticv1.Cluster {
	if len(constraint.Spec.Selector.Providers) == 0 {
		return ctList
	}
	var filteredList []kubermaticv1.Cluster

	providersSet := sets.NewString(constraint.Spec.Selector.Providers...)

	for _, cluster := range ctList {
		name, err := provider.ClusterCloudProviderName(cluster.Spec.Cloud)
		if err == nil && providersSet.Has(name) {
			filteredList = append(filteredList, cluster)
		}
	}
	return filteredList
}
