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
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Key = "default"
)

type clusterNamesList []string

func getClusterNameFromNamespace(clusterNamespace string) string {
	var clusterName string
	temp := strings.Split(clusterNamespace, "-")
	if len(temp) == 2 {
		clusterName = temp[1]
	}
	return clusterName
}

func getClusterNamesForExistingConstraint(ctx context.Context, client ctrlruntimeclient.Client, constraint *kubermaticv1.Constraint) (clusterNamesList, error) {

	var existingClusterNames []string

	labelSelector, err := labels.Parse(fmt.Sprintf("%s=%s", Key, constraint.Name))
	if err != nil {
		return nil, err
	}

	constraintList := &kubermaticv1.ConstraintList{}

	err = client.List(ctx, constraintList, &ctrlruntimeclient.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}

	for _, defaultConstraint := range constraintList.Items {
		clusterName := getClusterNameFromNamespace(defaultConstraint.Namespace)
		existingClusterNames = append(existingClusterNames, clusterName)
	}
	return existingClusterNames, nil
}

func getClusterList(ctx context.Context, client ctrlruntimeclient.Client, clusterNames clusterNamesList) []kubermaticv1.Cluster {
	var existingClusterList []kubermaticv1.Cluster
	for _, clusterName := range clusterNames {
		cluster := &kubermaticv1.Cluster{}
		err := client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster)
		if err != nil {
			return nil
		}
		existingClusterList = append(existingClusterList, *cluster)
	}

	return existingClusterList
}

func (s clusterNamesList) contains(searchterm string) bool {
	for _, value := range s {
		if value == searchterm {
			return true
		}
	}
	return false
}

func (s clusterNamesList) difference(s2 clusterNamesList) []string {
	var result []string
	for _, value := range s {
		if !s2.contains(value) {
			result = append(result, value)
		}
	}
	return result
}

// GetClustersForConstraint gets clusters for the constraints by using the constraints selector to filter out unselected clusters
func GetClustersForConstraint(ctx context.Context, client ctrlruntimeclient.Client,
	constraint *kubermaticv1.Constraint, workerNamesLabelSelector labels.Selector) ([]kubermaticv1.Cluster, []kubermaticv1.Cluster, error) {

	var desiredClusterNames clusterNamesList

	desiredList, err := getDesiredClusterListForConstraint(ctx, client, constraint, workerNamesLabelSelector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed listing clusters: %w", err)
	}

	desiredClusterList := desiredList.Items

	existingClusterNames, err := getClusterNamesForExistingConstraint(ctx, client, constraint)
	if existingClusterNames == nil || err != nil {
		return desiredClusterList, desiredClusterList, nil
	}

	for _, cluster := range desiredClusterList {
		desiredClusterNames = append(desiredClusterNames, cluster.Name)
	}

	unwantedClusterNames := existingClusterNames.difference(desiredClusterNames)
	unwantedClusterList := getClusterList(ctx, client, unwantedClusterNames)

	return desiredClusterList, unwantedClusterList, nil
}

func getDesiredClusterListForConstraint(ctx context.Context,
	client ctrlruntimeclient.Client,
	constraint *kubermaticv1.Constraint,
	workerNamesLabelSelector labels.Selector) (*kubermaticv1.ClusterList, error) {
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
