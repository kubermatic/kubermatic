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
	"errors"
	"fmt"
	"strconv"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func getOrgIDForProject(project *kubermaticv1.Project) (uint, bool) {
	annotation, ok := project.GetAnnotations()[GrafanaOrgAnnotationKey]
	if !ok {
		return 0, false
	}

	id, err := strconv.ParseUint(annotation, 10, 32)
	if err != nil {
		return 0, false
	}

	return uint(id), true
}

// getOrgByProject gets the org ID from the project's annotation and
// then fetches the organization from Grafana. The function will also ensure
// that the org name matches the project name, in order to prevent any
// controller from accidentally reconciling content into the wrong org.
func getOrgByProject(ctx context.Context, grafanaClient *grafanasdk.Client, project *kubermaticv1.Project) (grafanasdk.Org, error) {
	orgID, ok := getOrgIDForProject(project)
	if !ok {
		return grafanasdk.Org{}, errors.New("project should have grafana org annotation set")
	}

	org, err := grafanaClient.GetOrgById(ctx, orgID)
	if err != nil {
		return grafanasdk.Org{}, fmt.Errorf("failed to get Grafana org: %w", err)
	}

	if !orgNameMatchesProject(project, org.Name) {
		return grafanasdk.Org{}, fmt.Errorf("Grafana org %d has invalid name (%q)", orgID, org.Name)
	}

	return org, nil
}

func GetGrafanaOrgUser(ctx context.Context, grafanaClient *grafanasdk.Client, orgID, uid uint) (*grafanasdk.OrgUser, error) {
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

func addUserToOrg(ctx context.Context, grafanaClient *grafanasdk.Client, org grafanasdk.Org, user *grafanasdk.User, role grafanasdk.RoleType) error {
	// checking if user already exists in the corresponding organization
	orgUser, err := GetGrafanaOrgUser(ctx, grafanaClient, org.ID, user.ID)
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
			return fmt.Errorf("unable to update grafana user role: %w (status: %s, message: %s)", err,
				ptr.Deref(status.Status, "no status"), ptr.Deref(status.Message, "no message"))
		}
	}

	return nil
}

func removeUserFromOrg(ctx context.Context, grafanaClient *grafanasdk.Client, org grafanasdk.Org, user *grafanasdk.User) error {
	status, err := grafanaClient.DeleteOrgUser(ctx, org.ID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to delete org user: %w (status: %s, message: %s)", err,
			ptr.Deref(status.Status, "no status"), ptr.Deref(status.Message, "no message"))
	}
	return nil
}

func ensureOrgUser(ctx context.Context, grafanaClient *grafanasdk.Client, project *kubermaticv1.Project, email string, role grafanasdk.RoleType) error {
	user, err := grafanaClient.LookupUser(ctx, email)
	if err != nil {
		return err
	}

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
		return fmt.Errorf("failed to add grafana user to org: %w (status: %s, message: %s)", err,
			ptr.Deref(status.Status, "no status"), ptr.Deref(status.Message, "no message"))
	}
	return nil
}

func addDashboards(ctx context.Context, log *zap.SugaredLogger, grafanaClient *grafanasdk.Client, configMap *corev1.ConfigMap) error {
	for key, data := range configMap.Data {
		var board grafanasdk.Board
		if err := json.Unmarshal([]byte(data), &board); err != nil {
			return fmt.Errorf("unable to unmarshal dashboard from ConfigMap %s/%s (%s): %w", configMap.Namespace, configMap.Name, key, err)
		}
		if status, err := grafanaClient.SetDashboard(ctx, board, grafanasdk.SetDashboardParams{Overwrite: true}); err != nil {
			log.Errorw("unable to set dashboard",
				zap.Error(err),
				"status", ptr.Deref(status.Status, "no status"),
				"message", ptr.Deref(status.Message, "no message"))
			return err
		}
	}
	return nil
}

func deleteDashboards(ctx context.Context, log *zap.SugaredLogger, grafanaClient *grafanasdk.Client, configMap *corev1.ConfigMap) error {
	for _, data := range configMap.Data {
		var board grafanasdk.Board
		if err := json.Unmarshal([]byte(data), &board); err != nil {
			return fmt.Errorf("unable to unmarshal dashboard: %w", err)
		}
		if board.UID == "" {
			log.Debugw("dashboard doesn't have UID set, skipping", "title", board.Title)
			continue
		}
		if status, err := grafanaClient.DeleteDashboardByUID(ctx, board.UID); err != nil {
			log.Errorw("unable to delete dashboard",
				zap.Error(err),
				"status", ptr.Deref(status.Status, "no status"),
				"message", ptr.Deref(status.Message, "no message"))
			return err
		}
	}
	return nil
}
