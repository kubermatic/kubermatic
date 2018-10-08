package monitoring

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateConfigMap(t *testing.T) {
	tests := []struct {
		name    string
		cluster *kubermaticv1.Cluster
		err     error
		cfgm    *corev1.ConfigMap
	}{
		{
			name: "successfully created",
			err:  nil,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nico1",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						DatacenterName: TestDC,
					},
				},
				Address: kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-nico1",
				},
			},
		},
		{
			name: "needs update",
			err:  nil,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nico1",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						DatacenterName: TestDC,
					},
				},
				Address: kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-nico1",
				},
			},
			cfgm: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "prometheus", Namespace: "cluster-nico1"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var objects []runtime.Object
			if test.cfgm != nil {
				objects = append(objects, test.cfgm)
			}
			controller := newTestController(objects, []runtime.Object{test.cluster})
			beforeActionCount := len(controller.kubeClient.(*fake.Clientset).Actions())

			data, err := controller.getClusterTemplateData(test.cluster)
			if err != nil {
				t.Fatal(err)
				return
			}

			if err := controller.ensureConfigMaps(test.cluster, data); err != nil {
				t.Errorf("failed to ensure ConfigMap: %v", err)
			}
			if test.cfgm != nil {
				if len(controller.kubeClient.(*fake.Clientset).Actions()) != beforeActionCount+1 ||
					controller.kubeClient.(*fake.Clientset).Actions()[beforeActionCount].GetVerb() != "update" {
					t.Error("client made a unknown call/calls when updating a ConfigMap", controller.kubeClient.(*fake.Clientset).Actions()[beforeActionCount:])
				}
			} else {
				if len(controller.kubeClient.(*fake.Clientset).Actions()) != beforeActionCount+1 {
					t.Error("client made more or less than 1 call to create a ConfigMap", controller.kubeClient.(*fake.Clientset).Actions()[beforeActionCount:])
				}
			}

		})
	}
}
