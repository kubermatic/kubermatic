/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rbac

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GroupRole Helper struct that holds a group and its assigned role.
type GroupRole struct {
	Group string
	Role  string
}

// gets the groups and assigned roles for the given project
// Combines the default groups and roles (viewers, editors...) with the one defined in the GroupProjectBindings.
func getGroupRolesList(ctx context.Context, client ctrlruntimeclient.Client, projectName string) ([]GroupRole, error) {
	var groupRoles []GroupRole

	projectReq, err := labels.NewRequirement("projectID", selection.Equals, []string{projectName})
	if err != nil {
		return nil, fmt.Errorf("failed to get construct project label selector: %w", err)
	}
	gbpList := &kubermaticv1.GroupProjectBindingList{}
	err = client.List(ctx, gbpList, &ctrlruntimeclient.ListOptions{LabelSelector: labels.NewSelector().Add(*projectReq)})
	if err != nil {
		return nil, fmt.Errorf("failed to get group project bindings: %w", err)
	}

	// add the default ones, for them the group name and the role are the same
	for _, group := range AllGroupsPrefixes {
		groupRoles = append(groupRoles, GroupRole{Group: group, Role: group})
	}

	// add groupProjectBinding ones
	for _, gbp := range gbpList.Items {
		if gbp.DeletionTimestamp == nil {
			groupRoles = append(groupRoles, GroupRole{Group: gbp.Spec.Group, Role: gbp.Spec.Role})
		}
	}

	return groupRoles, nil
}
