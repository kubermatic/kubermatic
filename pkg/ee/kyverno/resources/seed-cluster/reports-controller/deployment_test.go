//go:build ee

package reportscontrollerresources

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
				ReportsController: &kubermaticv1.KyvernoControllerSettings{
					Resources: &corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
				},
			},
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
