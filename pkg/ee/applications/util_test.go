/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2024 Kubermatic GmbH

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

package applications

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"

	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var clusterNamespace = "test"

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
			namespace: "test",
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
					NamespaceName: "test",
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
				},
			},
		},
		{
			name:      "case 2: fetching template data should fail when cluster cannot be fetched",
			namespace: clusterNamespace,
			seedClient: fake.
				NewClientBuilder().WithObjects().Build(),
			want:    nil,
			wantErr: ErrNoClusterForTemplating,
		},
		{
			name:      "case 3: fetching template data should fail when cluster has no valid version",
			namespace: "test",
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
						ControlPlane: semver.Semver("vinvalid"),
					},
				},
			}).Build(),
			want:    nil,
			wantErr: ErrNoClusterForTemplating,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTemplateData(context.TODO(), tt.seedClient, tt.namespace)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("getTemplateData() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("getTemplateData() got = %v, want %v", got, tt.want)
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
			wantErr: fmt.Errorf("failed to render template: "),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderValueTemplate(tt.values, &tt.templateData)
			if (err != nil) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Fatalf("getTemplateData() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("getTemplateData() got = %v, want %v", got, tt.want)
			}
		})
	}
}
