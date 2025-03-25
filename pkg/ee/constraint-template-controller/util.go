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

package constrainttemplatecontroller

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetClustersForConstraintTemplate gets clusters for the CT by using the CT selector to filter out unselected clusters.
func GetClustersForConstraintTemplate(ctx context.Context, client ctrlruntimeclient.Client,
	ct *kubermaticv1.ConstraintTemplate, workerNamesLabelSelector labels.Selector,
) (*kubermaticv1.ClusterList, error) {
	ctLabelSelector, err := metav1.LabelSelectorAsSelector(&ct.Spec.Selector.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("error converting Constraint Template label selector (%v) to a kubernetes selector: %w", ct.Spec.Selector.LabelSelector, err)
	}

	var selector labels.Selector
	ctReq, _ := ctLabelSelector.Requirements()
	selector = workerNamesLabelSelector.Add(ctReq...)

	clusterList := &kubermaticv1.ClusterList{}
	if err := client.List(ctx, clusterList, &ctrlruntimeclient.ListOptions{LabelSelector: selector}); err != nil {
		return nil, fmt.Errorf("failed listing clusters: %w", err)
	}

	clusterList.Items = filterProvider(ct, clusterList.Items)
	return clusterList, nil
}

func filterProvider(ct *kubermaticv1.ConstraintTemplate, ctList []kubermaticv1.Cluster) []kubermaticv1.Cluster {
	if len(ct.Spec.Selector.Providers) == 0 {
		return ctList
	}
	var filteredList []kubermaticv1.Cluster

	providersSet := sets.New[string](ct.Spec.Selector.Providers...)

	for _, cluster := range ctList {
		name, err := kubermaticv1helper.ClusterCloudProviderName(cluster.Spec.Cloud)
		if err == nil && providersSet.Has(name) {
			filteredList = append(filteredList, cluster)
		}
	}
	return filteredList
}
