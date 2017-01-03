package cluster

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubermatic/api"
)

func TestLoadServiceFile(t *testing.T) {
	cc := &clusterController{
		masterResourcesPath: "./fixtures/templates/",
	}
	c := &api.Cluster{
		Address: &api.ClusterAddress{
			NodePort: 1234,
		},
	}

	res, err := loadServiceFile(cc, c, "test")
	if err != nil {
		t.Fatal(err)
	}

	checkTestResult(t, "test-service-result", res)
}

func TestLoadIngressFile(t *testing.T) {
	cc := &clusterController{
		masterResourcesPath: "./fixtures/templates/",
		dc:                  "asdf-de-1",
		externalURL:         "de.example.com",
	}
	c := &api.Cluster{
		Metadata: api.Metadata{
			Name: "de-test-01",
		},
	}

	res, err := loadIngressFile(cc, c, "test")
	if err != nil {
		t.Fatal(err)
	}

	checkTestResult(t, "test-ingress-result", res)
}

func TestLoadDeploymentFile(t *testing.T) {
	cc := &clusterController{
		masterResourcesPath: "./fixtures/templates/",
		dc:                  "asdf-de-1",
	}
	c := &api.Cluster{
		Metadata: api.Metadata{
			Name: "de-test-01",
		},
	}

	res, err := loadDeploymentFile(cc, c, "test")
	if err != nil {
		t.Fatal(err)
	}

	checkTestResult(t, "test-dep-result", res)
}

func TestLoadPVCFile(t *testing.T) {
	cc := &clusterController{
		masterResourcesPath: "./fixtures/templates/",
	}
	c := &api.Cluster{}

	res, err := loadPVCFile(cc, c, "test")
	if err != nil {
		t.Fatal(err)
	}

	checkTestResult(t, "test-pvc-result", res)
}

func TestLoadApiserverFile(t *testing.T) {
	cc := &clusterController{
		masterResourcesPath: "./fixtures/templates/",
		overwriteHost:       "localhost",
	}
	c := &api.Cluster{
		Address: &api.ClusterAddress{
			NodePort: 1234,
		},
	}

	res, err := loadApiserver(cc, c, "test-api-server")
	if err != nil {
		t.Fatal(err)
	}

	checkTestResult(t, "test-api-server-dep-result", res)
}

func checkTestResult(t *testing.T, resFile string, testObj interface{}) {
	exp, err := ioutil.ReadFile(filepath.Join("./fixtures/templates", resFile+".json"))
	if err != nil {
		t.Fatal(err)
	}

	res, err := json.Marshal(testObj)
	if err != nil {
		t.Fatal(err)
	}

	resStr := strings.TrimSpace(string(res))
	expStr := strings.TrimSpace(string(exp))

	if resStr != expStr {
		t.Fatalf("Expected to get %v, got %v", expStr, resStr)
	}
}
