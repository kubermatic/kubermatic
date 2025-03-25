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

package seedconstraintsynchronizer

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

const (
	Key = "default"
)

// FilterClustersForConstraint gets clusters for the constraints by using the constraints selector to filter out unselected clusters.
func FilterClustersForConstraint(ctx context.Context, client ctrlruntimeclient.Client, constraint *kubermaticv1.Constraint, clusterList *kubermaticv1.ClusterList) ([]kubermaticv1.Cluster, []kubermaticv1.Cluster, error) {
	constraintLabelSelector, err := metav1.LabelSelectorAsSelector(&constraint.Spec.Selector.LabelSelector)
	if err != nil {
		return nil, nil, fmt.Errorf("error converting Constraint label selector (%v) to a kubernetes selector: %w", constraint.Spec.Selector.LabelSelector, err)
	}
	providersSet := sets.New[string](constraint.Spec.Selector.Providers...)

	var unwanted []kubermaticv1.Cluster
	var desired []kubermaticv1.Cluster

	for _, cluster := range clusterList.Items {
		if !constraintLabelSelector.Matches(labels.Set(cluster.Labels)) {
			unwanted = append(unwanted, cluster)
			continue
		}

		name, err := kubermaticv1helper.ClusterCloudProviderName(cluster.Spec.Cloud)

		if err != nil {
			return nil, nil, err
		}

		if providersSet.Len() != 0 && !providersSet.Has(name) {
			unwanted = append(unwanted, cluster)
			continue
		}
		desired = append(desired, cluster)
	}

	return desired, unwanted, nil
}
