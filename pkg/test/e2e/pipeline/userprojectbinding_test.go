//go:build e2e

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

package pipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onsi/gomega"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	// userResolvedAnnotation mirrors the unexported literal in the UPB controller
	// (pkg/controller/master-controller-manager/user-project-binding/controller.go).
	// It is not exported, so the test hardcodes the same string.
	// TODO (burak): export this.
	userResolvedAnnotation = "userprojectbinding.kubermatic.k8c.io/user-resolved"

	testProjectName = "pipeline-upb"
	testUserEmail   = "pipeline-upb-user@example.com"
	testUserName    = "pipeline-upb-user"

	interval        = 1 * time.Second
	shortWait       = 30 * time.Second
	convergeAll     = 1 * time.Minute
	baselineCleanup = 2 * time.Minute
)

// TestUserProjectBinding asserts the PR #16131 behavior against a running KKP:
// a UserProjectBinding whose User does not exist yet must stay pending (not
// deleted); once the User appears it must converge (owner refs, finalizer, RBAC);
// and once a previously-resolved User is deleted the binding must be cleaned up.
func TestUserProjectBinding(t *testing.T) {
	client := getClient(t)

	// the project object name is the projectID the binding and RBAC aggregator key
	// off, so thread one lowercase value through everywhere. It must be a valid
	// RFC 1123 subdomain (GenProject's default "<name>-ID" suffix is uppercase and
	// the apiserver rejects it), and every derived RBAC name must stay valid too.
	projectID := testProjectName
	// GenBinding sets Spec.Group = "owners-<projectID>" and expects the bare prefix.
	ownerGroup := rbac.GenerateActualGroupNameFor(projectID, rbac.OwnerGroupNamePrefix)
	binding := generator.GenBinding(projectID, testUserEmail, rbac.OwnerGroupNamePrefix)
	// GenBinding names the object "<projectID>-<email>-owners"; the email's "@" and "."
	// are not valid in an RFC 1123 resource name, so override it with a valid name.
	// Spec (UserEmail/ProjectID/Group) stays as GenBinding set it.
	binding.Name = fmt.Sprintf("%s-owners", projectID)
	user := generator.GenUser("", testUserName, testUserEmail)

	feature := features.New("UserProjectBinding pending-delete").
		Setup(func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			g := gomega.NewWithT(t)

			assertCleanBaseline(ctx, g, client, projectID, binding.Name, user.Name)

			project := generator.GenProject(testProjectName, kubermaticv1.ProjectActive, generator.DefaultCreationTimestamp())
			project.Name = projectID
			g.Expect(client.Create(ctx, project)).To(gomega.Succeed(), "failed to create project")

			g.Eventually(func(g gomega.Gomega) {
				project := &kubermaticv1.Project{}
				g.Expect(client.Get(ctx, types.NamespacedName{Name: projectID}, project)).To(gomega.Succeed())
				g.Expect(project.Status.Phase).To(gomega.Equal(kubermaticv1.ProjectActive))
			}).WithContext(ctx).WithTimeout(convergeAll).WithPolling(interval).Should(gomega.Succeed(),
				"project never became Active")

			return ctx
		}).
		Assess("pending binding is not deleted", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			g := gomega.NewWithT(t)

			g.Expect(client.Create(ctx, binding)).To(gomega.Succeed(), "failed to create binding")

			g.Consistently(func(g gomega.Gomega) {
				got := &kubermaticv1.UserProjectBinding{}
				g.Expect(client.Get(ctx, types.NamespacedName{Name: binding.Name}, got)).To(gomega.Succeed(),
					"binding disappeared while User was absent (regression)")
				g.Expect(got.Annotations).NotTo(gomega.HaveKeyWithValue(userResolvedAnnotation, "true"),
					"binding was marked user-resolved before any User exists")
			}).WithContext(ctx).WithTimeout(shortWait).WithPolling(interval).Should(gomega.Succeed())

			return ctx
		}).
		Assess("converges once user exists", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			g := gomega.NewWithT(t)

			g.Expect(client.Create(ctx, user)).To(gomega.Succeed(), "failed to create user")

			roleName := fmt.Sprintf("kubermatic:userprojectbinding-%s:%s", binding.Name, ownerGroup)

			g.Eventually(func(g gomega.Gomega) {
				got := &kubermaticv1.UserProjectBinding{}
				g.Expect(client.Get(ctx, types.NamespacedName{Name: binding.Name}, got)).To(gomega.Succeed())
				g.Expect(got.Annotations).To(gomega.HaveKeyWithValue(userResolvedAnnotation, "true"))
				g.Expect(got.OwnerReferences).To(gomega.ContainElement(gomega.HaveField("Kind", kubermaticv1.ProjectKindName)),
					"binding missing Project owner reference")
				g.Expect(got.Finalizers).To(gomega.ContainElement(rbac.CleanupFinalizerName),
					"binding missing rbac cleanup finalizer")

				project := &kubermaticv1.Project{}
				g.Expect(client.Get(ctx, types.NamespacedName{Name: projectID}, project)).To(gomega.Succeed())
				g.Expect(project.OwnerReferences).To(gomega.ContainElement(gomega.HaveField("Kind", kubermaticv1.UserKindName)),
					"project missing User owner reference")

				clusterRole := &rbacv1.ClusterRole{}
				g.Expect(client.Get(ctx, types.NamespacedName{Name: roleName}, clusterRole)).To(gomega.Succeed())
				g.Expect(clusterRole.Labels).To(gomega.HaveKeyWithValue(kubermaticv1.AuthZRoleLabel, ownerGroup))

				clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
				g.Expect(client.Get(ctx, types.NamespacedName{Name: roleName}, clusterRoleBinding)).To(gomega.Succeed())
				g.Expect(clusterRoleBinding.Subjects).To(gomega.ContainElement(gomega.And(
					gomega.HaveField("Kind", rbacv1.GroupKind),
					gomega.HaveField("Name", ownerGroup),
				)), "ClusterRoleBinding missing Group subject")
			}).WithContext(ctx).WithTimeout(convergeAll).WithPolling(interval).Should(gomega.Succeed(),
				"binding did not converge after User creation")

			return ctx
		}).
		Assess("genuine orphan is deleted", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			g := gomega.NewWithT(t)

			err := client.Delete(ctx, user)
			g.Expect(ctrlruntimeclient.IgnoreNotFound(err)).To(gomega.Succeed(), "failed to delete user")

			g.Eventually(func(g gomega.Gomega) {
				got := &kubermaticv1.UserProjectBinding{}
				err := client.Get(ctx, types.NamespacedName{Name: binding.Name}, got)
				g.Expect(apierrors.IsNotFound(err)).To(gomega.BeTrue(),
					"binding still exists after its User was deleted")
			}).WithContext(ctx).WithTimeout(convergeAll).WithPolling(interval).Should(gomega.Succeed())

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			// best-effort cleanup; deleting the Project cascades its group subjects.
			bestEffortDelete(ctx, client, &kubermaticv1.UserProjectBinding{ObjectMeta: metav1.ObjectMeta{Name: binding.Name}})
			bestEffortDelete(ctx, client, &kubermaticv1.User{ObjectMeta: metav1.ObjectMeta{Name: user.Name}})
			bestEffortDelete(ctx, client, &kubermaticv1.Project{ObjectMeta: metav1.ObjectMeta{Name: projectID}})
			return ctx
		}).
		Feature()

	testEnv.Test(t, feature)
}

func getClient(t *testing.T) ctrlruntimeclient.Client {
	t.Helper()
	client, _ := utils.GetClientsOrDie()
	return client
}

// assertCleanBaseline converges the cluster to a clean state before the test
// creates anything. Previous runs can leave objects behind because the Project
// carries KKP cleanup finalizers and teardown does not wait for them to clear.
// Each poll re-checks every object, deletes any that still exist (re-issuing
// the delete when a finalizer stalls), and only succeeds once all are gone.
func assertCleanBaseline(ctx context.Context, g gomega.Gomega, client ctrlruntimeclient.Client, projectID, bindingName, userName string) {
	checks := []struct {
		obj  ctrlruntimeclient.Object
		name string
	}{
		{&kubermaticv1.Project{}, projectID},
		{&kubermaticv1.UserProjectBinding{}, bindingName},
		{&kubermaticv1.User{}, userName},
	}

	g.Eventually(func(g gomega.Gomega) {
		for _, c := range checks {
			err := client.Get(ctx, types.NamespacedName{Name: c.name}, c.obj)
			if apierrors.IsNotFound(err) {
				continue
			}
			g.Expect(err).To(gomega.Succeed(), "failed to get leftover %T %q", c.obj, c.name)

			// object still exists: (re)issue a delete so finalizers keep draining.
			_ = client.Delete(ctx, c.obj)

			// fail this poll so Eventually retries until the object is gone.
			g.Expect(apierrors.IsNotFound(err)).To(gomega.BeTrue(),
				"leftover %T %q still present, waiting for delete to drain", c.obj, c.name)
		}
	}).WithContext(ctx).WithTimeout(baselineCleanup).WithPolling(interval).Should(gomega.Succeed(),
		"leftover objects from a previous run never cleaned up")
}

func bestEffortDelete(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object) {
	_ = client.Delete(ctx, obj)
}
