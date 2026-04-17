//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2026 Kubermatic GmbH

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

package backgroundcontrollerresources

import (
	"testing"

	"github.com/stretchr/testify/require"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestDeploymentReconcilerUsesConfiguredResources(t *testing.T) {
	cluster := &kubermaticv1.Cluster{
		Spec: kubermaticv1.ClusterSpec{
			Kyverno: &kubermaticv1.KyvernoSettings{
				Enabled: true,
				BackgroundController: &kubermaticv1.KyvernoControllerSettings{
					Resources: &corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
				},
			},
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-namespace",
		},
	}

	data := resources.NewTemplateDataBuilder().
		WithCluster(cluster).
		Build()

	_, reconcile := DeploymentReconciler(data)()
	deployment, err := reconcile(&appsv1.Deployment{})
	require.NoError(t, err)

	require.Len(t, deployment.Spec.Template.Spec.Containers, 1)
	container := deployment.Spec.Template.Spec.Containers[0]

	require.Equal(t, "100m", container.Resources.Requests.Cpu().String())
	require.Equal(t, "64Mi", container.Resources.Requests.Memory().String())
	require.Equal(t, "256Mi", container.Resources.Limits.Memory().String())
}
