/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package monitoring

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
			ctx := context.Background()
			objects := []ctrlruntimeclient.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resources.ApiserverServiceName,
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

			data, err := controller.getClusterTemplateData(ctx, controller, test.cluster)
			if err != nil {
				t.Fatal(err)
				return
			}

			if err := controller.ensureConfigMaps(ctx, test.cluster, data); err != nil {
				t.Errorf("failed to ensure ConfigMap: %v", err)
			}

			keyName := types.NamespacedName{Namespace: test.cluster.Status.NamespaceName, Name: resources.PrometheusConfigConfigMapName}
			gotConfigMap := &corev1.ConfigMap{}
			if err := controller.Get(ctx, keyName, gotConfigMap); err != nil {
				t.Fatalf("failed to get the ConfigMap from the dynamic client: %v", err)
			}

			// Simply check if it has been overwritten. Doing a full comparison is not helpful here as we do that already in pkg/resources/test
			if gotConfigMap.Data["prometheus.yaml"] == "foo" {
				t.Error("expected key 'prometheus.yaml' did not get overwritten when it should have been")
			}
		})
	}
}
