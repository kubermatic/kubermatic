//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2025 Kubermatic GmbH

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

package cleanupcontrollerresources

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	commonseedresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/common"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourcesForDeletion returns a list of resources that should be deleted when the Kyverno cleanup controller is removed.
func ResourcesForDeletion(cluster *kubermaticv1.Cluster) []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      commonseedresources.KyvernoCleanupControllerDeploymentName,
				Namespace: cluster.Status.NamespaceName,
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      commonseedresources.KyvernoCleanupControllerServiceName,
				Namespace: cluster.Status.NamespaceName,
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      commonseedresources.KyvernoCleanupControllerMetricsServiceName,
				Namespace: cluster.Status.NamespaceName,
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      commonseedresources.KyvernoCleanupControllerRoleName,
				Namespace: cluster.Status.NamespaceName,
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      commonseedresources.KyvernoCleanupControllerRoleBindingName,
				Namespace: cluster.Status.NamespaceName,
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      commonseedresources.KyvernoCleanupControllerServiceAccountName,
				Namespace: cluster.Status.NamespaceName,
			},
		},
	}
}

// CleanUpResources deletes all resources created for the Kyverno cleanup controller.
func CleanUpResources(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	resources := ResourcesForDeletion(cluster)
	for _, resource := range resources {
		if err := client.Delete(ctx, resource); err != nil {
			return err
		}
	}
	return nil
}
