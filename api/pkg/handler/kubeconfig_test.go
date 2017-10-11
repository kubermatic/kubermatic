package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghodss/yaml"
	"k8s.io/client-go/tools/clientcmd/api/v1"
)

func TestKubeConfigEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/fake-1/cluster/234jkh24234g/kubeconfig", nil)

	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(false))

	e.ServeHTTP(res, req)
	checkStatusCode(http.StatusOK, res, t)

	b, err := yaml.YAMLToJSON(res.Body.Bytes())
	if err != nil {
		t.Error(err)
		return
	}

	var c *v1.Config
	if err := json.Unmarshal(b, &c); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if c.CurrentContext != "234jkh24234g" {
		t.Errorf("Expected response to be the default fake cluster, got %+v", c)
	}
}

func TestKubeConfigEndpointNotExistingDC(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/testtest/cluster/234jkh24234g/kubeconfig", nil)

	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(false))

	e.ServeHTTP(res, req)
	checkStatusCode(http.StatusBadRequest, res, t)
}

func TestKubeConfigEndpointNotExistingCluster(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/fake-1/cluster/testtest/kubeconfig", nil)

	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(false))

	e.ServeHTTP(res, req)
	checkStatusCode(http.StatusNotFound, res, t)
}
