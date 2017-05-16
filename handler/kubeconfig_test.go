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

	if res.Code != http.StatusOK {
		t.Errorf("Expected status code to be 200, got %d", res.Code)
		t.Error(res.Body.String())
		return
	}

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

	if res.Code != http.StatusBadRequest {
		t.Errorf("Expected status code to be 400, got %d", res.Code)
		t.Error(res.Body.String())
		return
	}

	exp := "unknown kubernetes datacenter \"testtest\""
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}

func TestKubeConfigEndpointNotExistingCluster(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/fake-1/cluster/testtest/kubeconfig", nil)

	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(false))

	e.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Errorf("Expected status code to be 404, got %d", res.Code)
		t.Error(res.Body.String())
		return
	}

	exp := "cluster \"testtest\" in dc \"fake-1\" not found"
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}
