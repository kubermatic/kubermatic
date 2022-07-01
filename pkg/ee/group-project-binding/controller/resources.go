//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

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

package controller

import (
	"context"
	"fmt"
	"reflect"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func clusterRoleBindingCreator(binding kubermaticv1.GroupProjectBinding, clusterRole rbacv1.ClusterRole) reconciling.NamedClusterRoleBindingCreatorGetter {
	name := fmt.Sprintf("%s:%s", clusterRole.Name, binding.Name)
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			if crb.Labels == nil {
				crb.Labels = map[string]string{}
			}

			crb.Labels[kubermaticv1.AuthZGroupProjectBindingLabel] = binding.Name
			crb.Labels[kubermaticv1.AuthZRoleLabel] = binding.Spec.Role

			crb.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.GroupProjectBindingKind,
					Name:       binding.Name,
					UID:        binding.UID,
				},
			}
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     clusterRole.Name,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Group",
					Name:     binding.Spec.Group,
				},
			}

			return crb, nil
		}
	}
}

func roleBindingCreator(binding kubermaticv1.GroupProjectBinding, role rbacv1.Role) reconciling.NamedRoleBindingCreatorGetter {
	name := fmt.Sprintf("%s:%s", role.Name, binding.Name)
	return func() (string, reconciling.RoleBindingCreator) {
		return name, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			if rb.Labels == nil {
				rb.Labels = map[string]string{}
			}

			rb.Labels[kubermaticv1.AuthZGroupProjectBindingLabel] = binding.Name
			rb.Labels[kubermaticv1.AuthZRoleLabel] = binding.Spec.Role

			rb.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.GroupProjectBindingKind,
					Name:       binding.Name,
					UID:        binding.UID,
				},
			}
			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     role.Name,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Group",
					Name:     binding.Spec.Group,
				},
			}

			return rb, nil
		}
	}
}

type GroupProjectBindingPatchFunc func(binding *kubermaticv1.GroupProjectBinding)

func updateGroupProjectBinding(ctx context.Context, client ctrlruntimeclient.Client, binding *kubermaticv1.GroupProjectBinding, patch GroupProjectBindingPatchFunc) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(binding)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetch the current state of the binding
		if err := client.Get(ctx, key, binding); err != nil {
			return err
		}

		// modify it
		original := binding.DeepCopy()
		patch(binding)

		// save some work
		if reflect.DeepEqual(original, binding) {
			return nil
		}

		// generate patch and update the GroupProjectBinding
		return client.Patch(ctx, binding, ctrlruntimeclient.MergeFrom(original))
	})
}
