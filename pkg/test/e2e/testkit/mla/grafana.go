/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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
	"strings"
	"time"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// canaryDashboardUID is the uid of the "Nodes Overview" dashboard. Its presence
// in the project org shows that the predefined dashboards were synced.
const canaryDashboardUID = "13yQpUxiz"

// grafanaOrgID waits for the project to carry the Grafana org annotation and
// returns the org ID.
func grafanaOrgID(ctx context.Context, in Input) (uint, error) {
	var orgID uint

	err := wait.PollImmediate(ctx, time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		project := &kubermaticv1.Project{}
		if err := in.SeedClient.Get(ctx, types.NamespacedName{Name: in.Project.Name}, project); err != nil {
			return fmt.Errorf("failed to get project: %w", err), nil
		}

		value, ok := project.Annotations[mla.GrafanaOrgAnnotationKey]
		if !ok {
			return fmt.Errorf("project has no %s annotation yet", mla.GrafanaOrgAnnotationKey), nil
		}

		id, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("unable to parse org ID %q: %w", value, err)
		}

		orgID = uint(id)

		return nil, nil
	})
	if err != nil {
		return 0, fmt.Errorf("waiting for the Grafana org annotation: %w", err)
	}

	return orgID, nil
}

// withGrafanaOrg runs fn with a Grafana client scoped to the project org.
func withGrafanaOrg(ctx context.Context, in Input, fn func(client *grafanasdk.Client, org grafanasdk.Org) error) error {
	orgID, err := grafanaOrgID(ctx, in)
	if err != nil {
		return err
	}

	client, stop, err := newGrafanaClient(ctx, in)
	if err != nil {
		return err
	}
	defer stop()

	org, err := client.GetOrgById(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to get Grafana org %d: %w", orgID, err)
	}

	client.SetOrgIDHeader(org.ID)

	return fn(client, org)
}

// CheckGrafanaOrg verifies that the project got a Grafana org, named after the
// project, and that the org ID in the project annotation (which the "Open
// Grafana" link uses) points to it.
func CheckGrafanaOrg(ctx context.Context, in Input) error {
	return withGrafanaOrg(ctx, in, func(_ *grafanasdk.Client, org grafanasdk.Org) error {
		expected := fmt.Sprintf("%s-%s", in.Project.Spec.Name, in.Project.Name)
		if org.Name != expected {
			return fmt.Errorf("Grafana org %d is named %q, expected %q", org.ID, org.Name, expected)
		}

		return nil
	})
}

// datasourceTypes are the Grafana datasources KKP creates per MLA cluster.
var datasourceTypes = []string{mla.PrometheusType, "loki", "alertmanager"}

// CheckDatasources verifies that the cluster's Prometheus, Loki and
// Alertmanager datasources exist in the project org and pass Grafana's health
// check.
func CheckDatasources(ctx context.Context, in Input) error {
	orgID, err := grafanaOrgID(ctx, in)
	if err != nil {
		return err
	}

	conn, err := newGrafanaConn(ctx, in)
	if err != nil {
		return err
	}
	defer conn.stop()

	for _, dsType := range datasourceTypes {
		uid := fmt.Sprintf("%s-%s", dsType, in.Cluster.Name)

		err := wait.PollImmediate(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
			// the Alertmanager datasource has no backend health check in
			// Grafana, so query the tenant's Alertmanager through the
			// datasource proxy instead
			if dsType == "alertmanager" {
				var status map[string]any
				return conn.get(ctx, orgID, "/api/datasources/proxy/uid/"+uid+"/alertmanager/api/v2/status", &status), nil
			}

			health := struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			}{}
			if err := conn.get(ctx, orgID, "/api/datasources/uid/"+uid+"/health", &health); err != nil {
				return err, nil
			}

			if health.Status != "OK" {
				return fmt.Errorf("health is %s: %s", health.Status, health.Message), nil
			}

			return nil, nil
		})
		if err != nil {
			return fmt.Errorf("datasource %s: %w", uid, err)
		}
	}

	return nil
}

// CheckCanaryDashboard verifies that the predefined dashboards were synced into
// the project org.
func CheckCanaryDashboard(ctx context.Context, in Input) error {
	return withGrafanaOrg(ctx, in, func(client *grafanasdk.Client, _ grafanasdk.Org) error {
		err := wait.PollImmediate(ctx, time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
			_, _, err := client.GetDashboardByUID(ctx, canaryDashboardUID)
			return err, nil
		})
		if err != nil {
			return fmt.Errorf("waiting for Grafana dashboard %s: %w", canaryDashboardUID, err)
		}

		return nil
	})
}

// CheckGrafanaUser verifies that the KKP admin is a Grafana admin and an editor
// in the project org.
func CheckGrafanaUser(ctx context.Context, in Input) error {
	if in.GrafanaAdminEmail == "" {
		return errors.New("Input.GrafanaAdminEmail is required for this check")
	}

	return withGrafanaOrg(ctx, in, func(client *grafanasdk.Client, org grafanasdk.Org) error {
		user := grafanasdk.User{}

		err := wait.PollImmediate(ctx, time.Second, 2*time.Minute, func(ctx context.Context) (error, error) {
			var err error

			user, err = client.LookupUser(ctx, in.GrafanaAdminEmail)
			if err != nil {
				return fmt.Errorf("user does not exist in Grafana yet: %w", err), nil
			}

			if !user.IsGrafanaAdmin || user.OrgID != org.ID {
				return fmt.Errorf("user expected to be Grafana admin with org ID %d, got admin=%v org ID %d", org.ID, user.IsGrafanaAdmin, user.OrgID), nil
			}

			return nil, nil
		})
		if err != nil {
			return fmt.Errorf("waiting for Grafana user: %w", err)
		}

		orgUser, err := mla.GetGrafanaOrgUser(ctx, client, org.ID, user.ID)
		if err != nil {
			return fmt.Errorf("failed to get Grafana org user: %w", err)
		}
		if orgUser == nil {
			return fmt.Errorf("user %s is not in the project org", in.GrafanaAdminEmail)
		}

		if orgUser.Role != string(grafanasdk.ROLE_EDITOR) {
			return fmt.Errorf("org user expected to have role %s, got %s", grafanasdk.ROLE_EDITOR, orgUser.Role)
		}

		return nil
	})
}

// CheckGrafanaViewer adds a viewer to the project and verifies that Grafana
// gives the user the Viewer role in the project org, and removes the user from
// the org once the project membership is gone.
func CheckGrafanaViewer(ctx context.Context, in Input) error {
	suffix := rand.String(6)
	email := fmt.Sprintf("mla-e2e-viewer-%s@kubermatic.com", suffix)

	user := &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{Name: "mla-e2e-viewer-" + suffix},
		Spec:       kubermaticv1.UserSpec{Email: email, Name: "MLA e2e viewer"},
	}
	if err := in.SeedClient.Create(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	defer deleteIgnoreNotFound(in, user)

	binding := &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "mla-e2e-viewer-" + suffix},
		Spec: kubermaticv1.UserProjectBindingSpec{
			UserEmail: email,
			ProjectID: in.Project.Name,
			Group:     rbac.GenerateActualGroupNameFor(in.Project.Name, rbac.ViewerGroupNamePrefix),
		},
	}
	if err := in.SeedClient.Create(ctx, binding); err != nil {
		return fmt.Errorf("failed to create project binding: %w", err)
	}
	defer deleteIgnoreNotFound(in, binding)

	return withGrafanaOrg(ctx, in, func(client *grafanasdk.Client, org grafanasdk.Org) error {
		var grafanaUserID uint

		err := wait.PollImmediate(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
			grafanaUser, err := client.LookupUser(ctx, email)
			if err != nil {
				return fmt.Errorf("user not in Grafana yet: %w", err), nil
			}

			orgUser, err := mla.GetGrafanaOrgUser(ctx, client, org.ID, grafanaUser.ID)
			if err != nil {
				return err, nil
			}
			if orgUser == nil {
				return errors.New("user not in the project org yet"), nil
			}

			if orgUser.Role != string(grafanasdk.ROLE_VIEWER) {
				return fmt.Errorf("org user has role %s, expected %s", orgUser.Role, grafanasdk.ROLE_VIEWER), nil
			}

			grafanaUserID = grafanaUser.ID

			return nil, nil
		})
		if err != nil {
			return fmt.Errorf("viewer not provisioned: %w", err)
		}

		if err := in.SeedClient.Delete(ctx, binding); err != nil {
			return fmt.Errorf("failed to delete project binding: %w", err)
		}

		err = wait.PollImmediate(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
			// GetGrafanaOrgUser returns no user and no error when the user is
			// not in the org
			orgUser, err := mla.GetGrafanaOrgUser(ctx, client, org.ID, grafanaUserID)
			if err != nil {
				return err, nil
			}
			if orgUser != nil {
				return errors.New("user is still in the project org"), nil
			}

			return nil, nil
		})
		if err != nil {
			return fmt.Errorf("viewer not removed from the org: %w", err)
		}

		return nil
	})
}

// CheckCustomDashboard adds a dashboard ConfigMap to the MLA namespace and
// verifies that the dashboard shows up in the project org, and disappears once
// the ConfigMap is deleted. Every MLA org on the seed gets the dashboard while
// it exists.
func CheckCustomDashboard(ctx context.Context, in Input) error {
	suffix := rand.String(6)
	uid := "mla-e2e-" + suffix

	dashboard, err := json.Marshal(map[string]any{
		"uid":           uid,
		"title":         "MLA e2e " + suffix,
		"panels":        []any{},
		"schemaVersion": 39,
	})
	if err != nil {
		return err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-dashboards-e2e-" + suffix,
			Namespace: mlaNamespace,
		},
		Data: map[string]string{uid + ".json": string(dashboard)},
	}
	if err := in.SeedClient.Create(ctx, configMap); err != nil {
		return fmt.Errorf("failed to create dashboard ConfigMap: %w", err)
	}
	defer deleteIgnoreNotFound(in, configMap)

	return withGrafanaOrg(ctx, in, func(client *grafanasdk.Client, _ grafanasdk.Org) error {
		err := wait.PollImmediate(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
			_, _, err := client.GetDashboardByUID(ctx, uid)
			return err, nil
		})
		if err != nil {
			return fmt.Errorf("dashboard %s not synced: %w", uid, err)
		}

		if err := in.SeedClient.Delete(ctx, configMap); err != nil {
			return fmt.Errorf("failed to delete dashboard ConfigMap: %w", err)
		}

		err = wait.PollImmediate(ctx, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
			_, _, err := client.GetDashboardByUID(ctx, uid)
			if isGrafanaNotFound(err) {
				return nil, nil
			}

			return fmt.Errorf("dashboard %s still exists (last error: %w)", uid, err), nil
		})
		if err != nil {
			return fmt.Errorf("dashboard not removed: %w", err)
		}

		return nil
	})
}

// deleteIgnoreNotFound is a best-effort cleanup for objects a check created.
func deleteIgnoreNotFound(in Input, obj ctrlruntimeclient.Object) {
	ctx, cancel := revertContext()
	defer cancel()

	if err := in.SeedClient.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		in.Log.Warnw("Failed to clean up", "object", obj.GetName(), zap.Error(err))
	}
}

// isGrafanaNotFound reports whether err is a Grafana 404. The SDK returns its
// ErrNotFound type only for some endpoints; GetDashboardByUID returns a plain
// error with the status code.
func isGrafanaNotFound(err error) bool {
	if err == nil {
		return false
	}

	return errors.As(err, &grafanasdk.ErrNotFound{}) || strings.Contains(err.Error(), "HTTP error 404")
}
