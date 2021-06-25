/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package mla

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/grafana/grafana/pkg/models"
	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func getOrgByProject(ctx context.Context, grafanaClient *grafanasdk.Client, project *kubermaticv1.Project) (grafanasdk.Org, error) {
	orgID, ok := project.GetAnnotations()[grafanaOrgAnnotationKey]
	if !ok {
		return grafanasdk.Org{}, fmt.Errorf("project should have grafana org annotation set")
	}
	id, err := strconv.ParseUint(orgID, 10, 32)
	if err != nil {
		return grafanasdk.Org{}, err
	}
	return grafanaClient.GetOrgById(ctx, uint(id))
}

func getGrafanaOrgUser(ctx context.Context, grafanaClient *grafanasdk.Client, orgID, uid uint) (*grafanasdk.OrgUser, error) {
	users, err := grafanaClient.GetOrgUsers(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.ID == uid {
			return &user, nil
		}
	}
	return nil, nil
}

func addUserToOrg(ctx context.Context, grafanaClient *grafanasdk.Client, org grafanasdk.Org, user *grafanasdk.User, role models.RoleType) error {
	// checking if user already exists in the corresponding organization
	orgUser, err := getGrafanaOrgUser(ctx, grafanaClient, org.ID, user.ID)
	if err != nil {
		return fmt.Errorf("unable to get user : %w", err)
	}
	// if there is no such user in project organization, let's add one
	if orgUser == nil {
		if err := addGrafanaOrgUser(ctx, grafanaClient, org.ID, user.Email, string(role)); err != nil {
			return fmt.Errorf("unable to add grafana user : %w", err)
		}
		return nil
	}

	if orgUser.Role != string(role) {
		userRole := grafanasdk.UserRole{
			LoginOrEmail: user.Email,
			Role:         string(role),
		}
		if status, err := grafanaClient.UpdateOrgUser(ctx, userRole, org.ID, orgUser.ID); err != nil {
			return fmt.Errorf("unable to update grafana user role: %w (status: %s, message: %s)", err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
		}
	}

	return nil
}

func removeUserFromOrg(ctx context.Context, grafanaClient *grafanasdk.Client, org grafanasdk.Org, user *grafanasdk.User) error {
	status, err := grafanaClient.DeleteOrgUser(ctx, org.ID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to delete org user: %w (status: %s, message: %s)", err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
	}
	return nil
}

func ensureOrgUser(ctx context.Context, grafanaClient *grafanasdk.Client, project *kubermaticv1.Project, userProjectBinding *kubermaticv1.UserProjectBinding) error {
	user, err := grafanaClient.LookupUser(ctx, userProjectBinding.Spec.UserEmail)
	if err != nil {
		return err
	}

	group := rbac.ExtractGroupPrefix(userProjectBinding.Spec.Group)
	role := groupToRole[group]

	org, err := getOrgByProject(ctx, grafanaClient, project)
	if err != nil {
		return err
	}

	return addUserToOrg(ctx, grafanaClient, org, &user, role)
}

func addGrafanaOrgUser(ctx context.Context, grafanaClient *grafanasdk.Client, orgID uint, email string, role string) error {
	userRole := grafanasdk.UserRole{
		LoginOrEmail: email,
		Role:         role,
	}
	if status, err := grafanaClient.AddOrgUser(ctx, userRole, orgID); err != nil {
		return fmt.Errorf("failed to add grafana user to org: %w (status: %s, message: %s)", err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
	}
	return nil
}

func addDashboards(ctx context.Context, log *zap.SugaredLogger, grafanaClient *grafanasdk.Client, configMap corev1.ConfigMap) error {
	for _, data := range configMap.Data {
		var board grafanasdk.Board
		if err := json.Unmarshal([]byte(data), &board); err != nil {
			return fmt.Errorf("unable to unmarshal dashboard: %w", err)
		}
		if status, err := grafanaClient.SetDashboard(ctx, board, grafanasdk.SetDashboardParams{Overwrite: true}); err != nil {
			log.Debugf("unable to set dashboard: %w (status: %s, message: %s)",
				err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
			return err
		}
	}
	return nil
}
