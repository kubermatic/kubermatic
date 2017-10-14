package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"encoding/json"
	"github.com/go-test/deep"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestClusterEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		clusterName  string
		responseCode int
		cluster      *kubermaticv1.Cluster
	}{
		{
			name:        "successful got cluster",
			clusterName: "foo",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: &kubermaticv1.ClusterAddress{
					AdminToken: "admintoken",
					URL:        "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{},
			},
			responseCode: http.StatusOK,
		},
		{
			name:        "unauthorized",
			clusterName: "foo",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": "not-current-user"},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: &kubermaticv1.ClusterAddress{
					AdminToken: "admintoken",
					URL:        "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{},
			},
			responseCode: http.StatusUnauthorized,
		},
		{
			name:        "not-found",
			clusterName: "not-existing-cluster",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: &kubermaticv1.ClusterAddress{
					AdminToken: "admintoken",
					URL:        "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{},
			},
			responseCode: http.StatusNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/cluster/"+test.clusterName, nil)
			res := httptest.NewRecorder()
			e := createTestEndpoint(getUser(false), []runtime.Object{test.cluster}, nil, nil)
			e.ServeHTTP(res, req)
			checkStatusCode(test.responseCode, res, t)

			if test.responseCode != http.StatusOK {
				return
			}

			gotCluster := &kubermaticv1.Cluster{}
			err := json.Unmarshal(res.Body.Bytes(), gotCluster)
			if err != nil {
				t.Fatal(err)
			}

			if diff := deep.Equal(gotCluster, test.cluster); diff != nil {
				t.Errorf("got different cluster than expected. Diff: %v", diff)
			}
		})
	}
}
