package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghodss/yaml"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	cmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

func TestKubeConfigEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/cluster/foo/kubeconfig", nil)

	res := httptest.NewRecorder()
	cluster := &kubermaticv1.Cluster{
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
	}
	e := createTestEndpoint(getUser(testUsername, false), []runtime.Object{cluster}, nil, nil)

	e.ServeHTTP(res, req)
	checkStatusCode(http.StatusOK, res, t)

	b, err := yaml.YAMLToJSON(res.Body.Bytes())
	if err != nil {
		t.Error(err)
		return
	}

	var c *cmdv1.Config
	if err := json.Unmarshal(b, &c); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if c.Clusters[0].Name != cluster.Name {
		t.Error("kubeconfig Clusters[0].Name is wrong")
	}

	if c.Clusters[0].Cluster.Server != cluster.Address.URL {
		t.Error("kubeconfig Clusters[0].Cluster.Server is wrong")
	}

	if string(c.Clusters[0].Cluster.CertificateAuthorityData) != string(cluster.Status.RootCA.Cert) {
		t.Error("kubeconfig Clusters[0].Cluster.CertificateAuthorityData is wrong")
	}

	if c.CurrentContext != cluster.Name {
		t.Error("kubeconfig current-context is wrong")
	}

	if c.AuthInfos[0].Name != cluster.Name {
		t.Error("kubeconfig AuthInfos[0].Name is wrong")
	}

	if c.AuthInfos[0].AuthInfo.Token != cluster.Address.AdminToken {
		t.Error("kubeconfig AuthInfos[0].AuthInfo.Token is wrong")
	}
}
