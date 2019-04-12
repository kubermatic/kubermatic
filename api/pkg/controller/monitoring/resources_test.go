package monitoring

import (
	"context"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func TestCreateConfigMap(t *testing.T) {
	tests := []struct {
		name      string
		cluster   *kubermaticv1.Cluster
		err       error
		configMap *corev1.ConfigMap
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
					Version: *semver.NewSemverOrDie("v1.11.3"),
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
					Version: *semver.NewSemverOrDie("v1.11.3"),
				},
				Address: kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-nico1",
				},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resources.PrometheusConfigConfigMapName,
					Namespace: "cluster-nico1",
				},
				Data: map[string]string{
					"foo": "bar",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			objects := []runtime.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resources.ApiserverExternalServiceName,
						Namespace: "cluster-nico1",
					},
					Spec: corev1.ServiceSpec{
						Ports:     []corev1.ServicePort{{NodePort: 99}},
						ClusterIP: "192.0.2.10",
					},
				},
				test.cluster,
			}
			if test.configMap != nil {
				objects = append(objects, test.configMap)
			}
			controller := newTestReconciler(t, objects)

			data, err := controller.getClusterTemplateData(context.Background(), controller.Client, test.cluster)
			if err != nil {
				t.Fatal(err)
				return
			}

			if err := controller.ensureConfigMaps(context.Background(), test.cluster, data); err != nil {
				t.Errorf("failed to ensure ConfigMap: %v", err)
			}

			keyName := types.NamespacedName{Namespace: test.cluster.Status.NamespaceName, Name: resources.PrometheusConfigConfigMapName}
			gotConfigMap := &corev1.ConfigMap{}
			if err := controller.Client.Get(context.Background(), keyName, gotConfigMap); err != nil {
				t.Fatalf("failed to get the ConfigMap from the dynamic client: %v", err)
			}

			// Simply check if it has been overwritten. Doing a full comparison is not helpful here as we do that already in pkg/resources/test
			if gotConfigMap.Data["prometheus.yaml"] == "foo" {
				t.Error("expected key 'prometheus.yaml' did not get overwritten when it should have been")
			}
		})
	}
}
