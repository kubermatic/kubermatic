/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package template

import (
	"context"
	"errors"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	clusterNamespace = "test-cluster"
	testObjectName   = "cluster-autoscaler"
	testObjectMeta   = metav1.ObjectMeta{
		Name:      testObjectName,
		Namespace: clusterNamespace,
	}
)

func TestGetTemplateData(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		seedClient ctrlruntimeclient.Client
		want       *TemplateData
		wantErr    error
	}{
		{
			name:      "case 1: fetching template data should succeed with a valid cluster resource",
			namespace: "test-cluster",
			seedClient: fake.
				NewClientBuilder().WithObjects(&kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster",
				},
				Spec: kubermaticv1.ClusterSpec{
					HumanReadableName: "humanreadable-cluster-name",
				},
				Status: kubermaticv1.ClusterStatus{
					UserEmail: "owner@email.com",
					Address: kubermaticv1.ClusterAddress{
						URL:  "https://cluster.seed.kkp",
						Port: 6443,
					},
					Versions: kubermaticv1.ClusterVersionsStatus{
						ControlPlane: semver.Semver("v1.30.5"),
					},
					NamespaceName: "test-cluster",
				},
			}).Build(),
			want: &TemplateData{
				ClusterData{
					Name:              "test-cluster",
					HumanReadableName: "humanreadable-cluster-name",
					OwnerEmail:        "owner@email.com",
					Address: kubermaticv1.ClusterAddress{
						URL:  "https://cluster.seed.kkp",
						Port: 6443,
					},
					Version:           "1.30.5",
					MajorMinorVersion: "1.30",
					AutoscalerVersion: "v1.30.3",
				},
			},
		},
		{
			name:      "case 2: fetching template data should fail when cluster cannot be fetched",
			namespace: clusterNamespace,
			seedClient: fake.
				NewClientBuilder().WithObjects().Build(),
			want:    nil,
			wantErr: apierrors.NewNotFound(schema.GroupResource{Group: "kubermatic.k8c.io", Resource: "clusters"}, "test-cluster"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTemplateData(context.Background(), tt.seedClient, tt.namespace)
			if err != nil && tt.wantErr == nil {
				t.Fatalf("Got unexpected error. %v", err)
			}
			if err == nil && tt.wantErr != nil {
				t.Fatalf("Got no error when one is expected. %v", tt.wantErr)
			}
			if apierrors.IsNotFound(tt.wantErr) && !apierrors.IsNotFound(err) {
				t.Fatalf("Got unexpected error. diff: %s, %s", err.Error(), tt.wantErr)
			}
			if changes := diff.ObjectDiff(got, tt.want); changes != "" {
				t.Fatalf("Got unexpected result. diff: %s", changes)
			}
		})
	}
}

func TestRenderValueTemplate(t *testing.T) {
	tests := []struct {
		name         string
		values       map[string]any
		templateData TemplateData
		want         map[string]any
		wantErr      error
	}{
		{
			name: "case 1: rendering for cluster name and version should succeed",
			values: map[string]any{
				"key1": "value1",
				"key2": "{{ .Cluster.Name }}",
				"key3": map[string]any{
					"nestedkey": "{{ .Cluster.Version }}",
				},
			},
			templateData: TemplateData{
				Cluster: ClusterData{
					Name:    "test",
					Version: "9.9.9",
				},
			},
			want: map[string]any{
				"key1": "value1",
				"key2": "test",
				"key3": map[string]any{
					"nestedkey": "9.9.9",
				},
			},
			wantErr: nil,
		},
		{
			name: "case 2: parsing unknown variables should lead to an error",
			values: map[string]any{
				"key1": "value1",
				"key2": "{{ .Foo.Name }}",
				"key3": map[string]any{
					"nestedkey": "{{ .Foo.Version }}",
				},
			},
			templateData: TemplateData{
				Cluster: ClusterData{
					Name:    "test",
					Version: "9.9.9",
				},
			},
			want:    nil,
			wantErr: ErrBadTemplate,
		},
		{
			name: "case 3: rendering helm values for the certain environment should succeed",
			values: map[string]any{
				"key1": "{{ .Cluster.Env }}",
			},
			templateData: TemplateData{
				Cluster: ClusterData{
					Env: "dev",
				},
			},
			want: map[string]any{
				"key1": "dev",
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderValueTemplate(tt.values, &tt.templateData)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Got unexpected error. diff: %v, %v", err, tt.wantErr)
			}

			if changes := diff.ObjectDiff(got, tt.want); changes != "" {
				t.Fatalf("Got unexpected result. diff: %s", changes)
			}
		})
	}
}

func TestHandleAddonCleanup(t *testing.T) {
	tests := []struct {
		name         string
		appName      string
		appNamespace string
		seedClient   ctrlruntimeclient.Client
		wantErr      error
		wantAPIErr   error
	}{
		{
			name:         "case 1: no error occurs when cleaning up the existing addon configured without reconciliation",
			appName:      "cluster-autoscaler",
			appNamespace: "test-app",
			seedClient: fake.
				NewClientBuilder().WithObjects(
				&kubermaticv1.Addon{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testObjectName,
						Namespace: clusterNamespace,
						Labels: map[string]string{
							"kubermatic-addon": "cluster-autoscaler",
						},
					},
				},
			).Build(),
			wantErr:    nil,
			wantAPIErr: apierrors.NewNotFound(schema.GroupResource{Group: "kubermatic.k8c.io", Resource: "addons"}, "cluster-autoscaler"),
		},
		{
			name:         "case 2: no error occurs when no addon needs to be cleaned up",
			appName:      "cluster-autoscaler",
			appNamespace: "test-app",
			seedClient: fake.
				NewClientBuilder().WithObjects().Build(),
			wantErr:    nil,
			wantAPIErr: apierrors.NewNotFound(schema.GroupResource{Group: "kubermatic.k8c.io", Resource: "addons"}, "cluster-autoscaler"),
		},
		{
			name:         "case 3: an error occurs when cleaning up the existing addon configured with reconciliation",
			appName:      "cluster-autoscaler",
			appNamespace: "test-app",
			seedClient: fake.
				NewClientBuilder().WithObjects(
				&kubermaticv1.Addon{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testObjectName,
						Namespace: clusterNamespace,
						Labels: map[string]string{
							"kubermatic-addon": "cluster-autoscaler",
							AddonEnforcedLabel: "true",
						},
					},
				},
			).Build(),
			wantErr:    ErrExistingAddon,
			wantAPIErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HandleAddonCleanup(context.Background(), tt.appName, clusterNamespace, tt.seedClient, kubermaticlog.Logger)
			if err != nil && tt.wantErr == nil {
				t.Fatalf("Got unexpected error. %v", err)
			}
			if err == nil && tt.wantErr != nil {
				t.Fatalf("Got no error when one is expected. %v", tt.wantErr)
			}

			serviceAccount := &kubermaticv1.Addon{
				ObjectMeta: testObjectMeta,
			}
			err = tt.seedClient.Get(context.Background(), ctrlruntimeclient.ObjectKeyFromObject(serviceAccount), serviceAccount)
			if err != nil {
				if compare := diff.StringDiff(tt.wantAPIErr.Error(), err.Error()); compare != "" {
					t.Fatalf("Got unexpected when fetching addon resource. %v", err)
				}
			}

			if err == nil && tt.wantAPIErr != nil {
				t.Fatalf("Got no error when one is expected from kube apiserver. %v", tt.wantAPIErr)
			}
		})
	}
}
