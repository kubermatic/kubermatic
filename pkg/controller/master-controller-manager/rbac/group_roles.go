package rbac

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/fields"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GroupRole Helper struct that holds a group and its assigned role
type GroupRole struct {
	Group string
	Role  string
}

// gets the groups and assigned roles for the given project
// Combines the default groups and roles (viwers, editors...) with the one defined in the GroupProjectBindings
func getGroupRolesList(ctx context.Context, client ctrlruntimeclient.Client, projectName string) ([]GroupRole, error) {
	var groupRoles []GroupRole

	gbpList := &kubermaticv1.GroupProjectBindingList{}
	err := client.List(ctx, gbpList, &ctrlruntimeclient.ListOptions{FieldSelector: fields.OneTermEqualSelector("projectID", projectName)})
	if err != nil {
		return nil, fmt.Errorf("failed to get group project bindings: %w", err)
	}

	// add the default ones, for them the group name and the role are the same
	for _, group := range AllGroupsPrefixes {
		groupRoles = append(groupRoles, GroupRole{Group: group, Role: group})
	}

	// add groupProjectBinding ones
	for _, gbp := range gbpList.Items {
		groupRoles = append(groupRoles, GroupRole{Group: gbp.Spec.Group, Role: gbp.Spec.ProjectGroup})
	}

	return groupRoles, nil
}
