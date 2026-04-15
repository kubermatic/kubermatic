//go:build ee

package admissioncontrollerresources

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
				AdmissionController: &kubermaticv1.KyvernoAdmissionControllerSettings{
					Resources: &corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("768Mi"),
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

	require.Len(t, deployment.Spec.Template.Spec.InitContainers, 1)
	initContainer := deployment.Spec.Template.Spec.InitContainers[0]
	require.Equal(t, "10m", initContainer.Resources.Requests.Cpu().String())
	require.Equal(t, "64Mi", initContainer.Resources.Requests.Memory().String())
	require.Equal(t, "100m", initContainer.Resources.Limits.Cpu().String())
	require.Equal(t, "256Mi", initContainer.Resources.Limits.Memory().String())

	require.Len(t, deployment.Spec.Template.Spec.Containers, 1)
	container := deployment.Spec.Template.Spec.Containers[0]
	require.Equal(t, "100m", container.Resources.Requests.Cpu().String())
	require.Equal(t, "128Mi", container.Resources.Requests.Memory().String())
	require.Equal(t, "768Mi", container.Resources.Limits.Memory().String())
}
