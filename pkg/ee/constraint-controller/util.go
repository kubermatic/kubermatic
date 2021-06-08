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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Key = "default"
)

type clusterNamesList []string

func getClusterList(ctx context.Context, client ctrlruntimeclient.Client, clusterNames clusterNamesList) []kubermaticv1.Cluster {
	var undesiredClusterList []kubermaticv1.Cluster
	for _, clusterName := range clusterNames {
		cluster := &kubermaticv1.Cluster{}
		err := client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster)
		if err != nil {
			return nil
		}
		undesiredClusterList = append(undesiredClusterList, *cluster)
	}

	return undesiredClusterList
}

func getClusterNames(clusterList *kubermaticv1.ClusterList) clusterNamesList {
	var clusterNames clusterNamesList
	for _, cluster := range clusterList.Items {
		clusterNames = append(clusterNames, cluster.Name)
	}
	return clusterNames
}

// FilterClustersForConstraint gets clusters for the constraints by using the constraints selector to filter out unselected clusters
func FilterClustersForConstraint(ctx context.Context, client ctrlruntimeclient.Client, constraint *kubermaticv1.Constraint, clusterList *kubermaticv1.ClusterList) (*kubermaticv1.ClusterList, *kubermaticv1.ClusterList, error) {
	var desiredClusterNames clusterNamesList
	var existingClusterNames clusterNamesList

	desiredList, err := getDesiredClusterListForConstraint(ctx, client, constraint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed listing clusters: %w", err)
	}

	existingClusterNames = getClusterNames(clusterList)
	desiredClusterNames = getClusterNames(desiredList)

	existing := sets.NewString(existingClusterNames...)
	desired := sets.NewString(desiredClusterNames...)
	unwantedClusterNames := existing.Difference(desired)

	undesiredClusterList := getClusterList(ctx, client, unwantedClusterNames.List())
	undesiredList := &kubermaticv1.ClusterList{}
	undesiredList.Items = undesiredClusterList

	return desiredList, undesiredList, nil
}

func getDesiredClusterListForConstraint(ctx context.Context,
	client ctrlruntimeclient.Client,
	constraint *kubermaticv1.Constraint) (*kubermaticv1.ClusterList, error) {

	constraintLabelSelector, err := v1.LabelSelectorAsSelector(&constraint.Spec.Selector.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("error converting Constraint label selector (%v) to a kubernetes selector: %w", constraint.Spec.Selector.LabelSelector, err)
	}

	clusterList := &kubermaticv1.ClusterList{}

	if err := client.List(ctx, clusterList, &ctrlruntimeclient.ListOptions{LabelSelector: constraintLabelSelector}); err != nil {
		return nil, fmt.Errorf("failed listing clusters: %w", err)
	}

	clusterList.Items = filterProvider(constraint, clusterList.Items)
	return clusterList, nil
}

func filterProvider(constraint *kubermaticv1.Constraint, clusterList []kubermaticv1.Cluster) []kubermaticv1.Cluster {
	if len(constraint.Spec.Selector.Providers) == 0 {
		return clusterList
	}
	var filteredList []kubermaticv1.Cluster

	providersSet := sets.NewString(constraint.Spec.Selector.Providers...)

	for _, cluster := range clusterList {
		name, err := provider.ClusterCloudProviderName(cluster.Spec.Cloud)
		if err == nil && providersSet.Has(name) {
			filteredList = append(filteredList, cluster)
		}
	}
	return filteredList
}
