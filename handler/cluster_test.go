package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"bytes"

	"github.com/kubermatic/api"
)

func TestNewClusterEndpoint(t *testing.T) {
	reqObj := &api.Cluster{
		Spec: api.ClusterSpec{
			HumanReadableName: "test-cluster",
		},
	}

	req := httptest.NewRequest("POST", "/api/v1/dc/fake-1/cluster", encodeReq(t, reqObj))
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()
	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 200, got %d", res.Code)
	}

	var c api.Cluster
	if err := json.Unmarshal(res.Body.Bytes(), &c); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if c.Metadata.UID == "" {
		t.Error("Expected cluster UID to be filled, got nil.")
	}
	if c.Metadata.Name == "" {
		t.Error("Expected cluster name to be filled, got nil.")
	}
	if c.Spec.HumanReadableName != reqObj.Spec.HumanReadableName {
		t.Errorf("Expected cluster name to be %s, got %s.", reqObj.Spec.HumanReadableName, c.Spec.HumanReadableName)
	}
}

func TestNewClusterEndpointV2(t *testing.T) {
	body := []byte("{\"cloud\":{\"user\":\"secret\",\"secret\":\"evenmoresecret\",\"name\":\"fake\",\"region\":\"fake-1\"},\"spec\":{\"humanReadableName\":\"test-1\"},\"ssh_keys\":[\"test\"]}")

	req := httptest.NewRequest("POST", "/api/v1/cluster", bytes.NewReader(body))
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()
	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 200, got %d", res.Code)
	}

	var c api.Cluster
	if err := json.Unmarshal(res.Body.Bytes(), &c); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if c.Metadata.UID == "" {
		t.Error("Expected cluster UID to be filled, got nil.")
	}
	if c.Metadata.Name == "" {
		t.Error("Expected cluster name to be filled, got nil.")
	}
	if c.Spec.HumanReadableName != "test-1" {
		t.Errorf("Expected cluster name to be %s, got %s.", "test-1", c.Spec.HumanReadableName)
	}
	if c.Spec.Cloud == nil {
		t.Fatal("Expected Cluster.Spec.Cloud to be filled, got nil")
	}
	if c.Spec.Cloud == nil {
		t.Fatal("Expected Cluster.Spec.Cloud to be filled, got nil")
	}
	if c.Spec.Cloud.Fake == nil {
		t.Fatal("Expected Cluster.Spec.Cloud.Fake to be filled, got nil")
	}
	if c.Spec.Cloud.Fake.Token != "evenmoresecret" {
		t.Fatalf("Expected Cluster.Spec.Cloud.Fake to be \"evenmoresecret\", got %q", c.Spec.Cloud.Fake.Token)
	}
}

func TestNewClusterEndpointNotExistingDC(t *testing.T) {
	reqObj := &api.Cluster{
		Spec: api.ClusterSpec{
			HumanReadableName: "test-cluster",
		},
	}

	req := httptest.NewRequest("POST", "/api/v1/dc/testtest/cluster", encodeReq(t, reqObj))
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()
	e.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 400, got %d", res.Code)
		t.Error(res.Body.String())
		return
	}

	exp := "unknown kubernetes datacenter \"testtest\"\n"
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}

func TestClustersEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/fake-1/cluster", nil)
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()

	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 200, got %d", res.Code)
		return
	}

	var cs []*api.Cluster
	if err := json.Unmarshal(res.Body.Bytes(), &cs); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if len(cs) != 1 {
		t.Errorf("Expected list of clusters to be of length 1, got length %d", len(cs))
		t.Error(res.Body.String())
	}
}

func TestClustersEndpointWithACreatedCluster(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/fake-1/cluster", nil)
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()

	cc, err := createTestCluster(t, e)
	if err != nil {
		return
	}

	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 200, got %d", res.Code)
		return
	}

	var cs []*api.Cluster
	if err := json.Unmarshal(res.Body.Bytes(), &cs); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if len(cs) != 2 {
		t.Errorf("Expected list of clusters to be of length 2, got length %d", len(cs))
		t.Error(res.Body.String())
		return
	}

	for _, c := range cs {
		if c.Metadata.UID == cc.Metadata.UID && c.Metadata.Name == cc.Metadata.Name {
			return
		}
	}

	t.Error("Expected cluster to contain the created one.")
}

func TestClusterEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/fake-1/cluster/234jkh24234g", nil)
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()

	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 200, got %d", res.Code)
		return
	}

	var c *api.Cluster
	if err := json.Unmarshal(res.Body.Bytes(), &c); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if c.Metadata.Name != "234jkh24234g" {
		t.Errorf("Expeced response to be the default fake cluster, got %+v", c)
	}
}

func TestClusterEndpointNotExistingDC(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/testtest/cluster/234jkh24234g", nil)
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()

	e.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 400, got %d", res.Code)
		return
	}

	exp := "unknown kubernetes datacenter \"testtest\"\n"
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}

func TestClusterEndpointNotExistingCluster(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dc/fake-1/cluster/testtest", nil)
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()

	e.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 404, got %d", res.Code)
		return
	}

	exp := "cluster \"testtest\" in dc \"fake-1\" not found\n"
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}

func TestSetCloudEndpointBringYourOwn(t *testing.T) {
	reqObj := &api.CloudSpec{
		BringYourOwn: &api.BringYourOwnCloudSpec{},
	}

	req := httptest.NewRequest("PUT", "/api/v1/dc/fake-1/cluster/234jkh24234g/cloud", encodeReq(t, reqObj))
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()

	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 200, got %d", res.Code)
		return
	}

	var c *api.Cluster
	if err := json.Unmarshal(res.Body.Bytes(), &c); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if c.Metadata.Name != "234jkh24234g" {
		t.Errorf("Expeced response to be the default fake cluster, got %+v", c)
	}
}

func TestSetCloudEndpointBringYourOwnNotExistingDC(t *testing.T) {
	reqObj := &api.CloudSpec{
		BringYourOwn: &api.BringYourOwnCloudSpec{},
	}

	req := httptest.NewRequest("PUT", "/api/v1/dc/testtest/cluster/234jkh24234g/cloud", encodeReq(t, reqObj))
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()

	e.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 400, got %d", res.Code)
		return
	}

	exp := "unknown kubernetes datacenter \"testtest\"\n"
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}

func TestSetCloudEndpointBringYourOwnNotExistingCluster(t *testing.T) {
	reqObj := &api.CloudSpec{
		BringYourOwn: &api.BringYourOwnCloudSpec{},
	}

	req := httptest.NewRequest("PUT", "/api/v1/dc/fake-1/cluster/testtest/cloud", encodeReq(t, reqObj))
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()

	e.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 404, got %d", res.Code)
		return
	}

	exp := "cluster \"testtest\" in dc \"fake-1\" not found\n"
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}

func TestSetCloudEndpointAWS(t *testing.T) {
	reqObj := &api.CloudSpec{
		DatacenterName: "fake-1",
		AWS:            &api.AWSCloudSpec{},
	}

	req := httptest.NewRequest("PUT", "/api/v1/dc/fake-1/cluster/234jkh24234g/cloud", encodeReq(t, reqObj))
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()

	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 200, got %d", res.Code)
		return
	}

	var c *api.Cluster
	if err := json.Unmarshal(res.Body.Bytes(), &c); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return
	}

	if c.Metadata.Name != "234jkh24234g" {
		t.Errorf("Expeced response to be the default fake cluster, got %+v", c)
	}
}

func TestSetCloudEndpointAWSNotExistingDC(t *testing.T) {
	reqObj := &api.CloudSpec{
		DatacenterName: "fake-1",
		AWS:            &api.AWSCloudSpec{},
	}

	req := httptest.NewRequest("PUT", "/api/v1/dc/testtest/cluster/234jkh24234g/cloud", encodeReq(t, reqObj))
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()

	e.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 400, got %d", res.Code)
		return
	}

	exp := "unknown kubernetes datacenter \"testtest\"\n"
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}

func TestSetCloudEndpointAWSNotExistingCluster(t *testing.T) {
	reqObj := &api.CloudSpec{
		DatacenterName: "fake-1",
		AWS:            &api.AWSCloudSpec{},
	}

	req := httptest.NewRequest("PUT", "/api/v1/dc/fake-1/cluster/testtest/cloud", encodeReq(t, reqObj))
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e := createTestEndpoint()

	e.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 404, got %d", res.Code)
		return
	}

	exp := "cluster \"testtest\" in dc \"fake-1\" not found\n"
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}

func TestDeleteCluster(t *testing.T) {
	res := httptest.NewRecorder()
	e := createTestEndpoint()
	tc, err := createTestCluster(t, e)
	if err != nil {
		return
	}

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/dc/fake-1/cluster/%s", tc.Metadata.Name), nil)
	authenticateHeader(req, false)

	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 200, got %d", res.Code)
		return
	}
}

func TestDeleteClusterNotExistingDC(t *testing.T) {
	res := httptest.NewRecorder()
	e := createTestEndpoint()
	tc, err := createTestCluster(t, e)
	if err != nil {
		return
	}

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/dc/testtest/cluster/%s", tc.Metadata.Name), nil)
	authenticateHeader(req, false)

	e.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 400, got %d", res.Code)
		return
	}

	exp := "unknown kubernetes datacenter \"testtest\"\n"
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}

func TestDeleteClusterNotExistingCluster(t *testing.T) {
	res := httptest.NewRecorder()
	e := createTestEndpoint()
	_, err := createTestCluster(t, e)
	if err != nil {
		return
	}

	req := httptest.NewRequest("DELETE", "/api/v1/dc/fake-1/cluster/testtest", nil)
	authenticateHeader(req, false)

	e.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 404, got %d", res.Code)
		return
	}

	exp := "cluster \"testtest\" in dc \"fake-1\" not found\n"
	if res.Body.String() != exp {
		t.Errorf("Expected error to be %q, got %q", exp, res.Body.String())
	}
}

func createTestCluster(t *testing.T, e http.Handler) (*api.Cluster, error) {
	reqObj := api.Cluster{
		Spec: api.ClusterSpec{
			HumanReadableName: "test-cluster",
		},
	}

	req := httptest.NewRequest("POST", "/api/v1/dc/fake-1/cluster", encodeReq(t, &reqObj))
	authenticateHeader(req, false)

	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Error(res.Body.String())
		t.Fatalf("Expected status code to be 200, got %d", res.Code)
		return nil, fmt.Errorf("Expected status code to be 200, got %d", res.Code)
	}

	var c *api.Cluster
	if err := json.Unmarshal(res.Body.Bytes(), &c); err != nil {
		t.Error(res.Body.String())
		t.Error(err)
		return nil, err
	}

	return c, nil
}
