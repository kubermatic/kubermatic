package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	crdclientfake "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
)

func createTestEndpoint(user auth.User) http.Handler {
	ctx := context.Background()
	dcs, err := provider.LoadDatacentersMeta("./fixtures/datacenters.yaml")
	if err != nil {
		log.Fatal(fmt.Printf("failed to load datacenter yaml %q: %v", "./fixtures/datacenters.yaml", err))
	}
	// create CloudProviders
	cps := cloud.Providers(dcs)
	kps, err := kubernetes.Providers("./fixtures/kubecfg.yaml", dcs, cps, "user1")
	if err != nil {
		log.Fatal(err)
	}

	// override the default master k8s provider since it would be a "real" k8s provider, not a fake one.
	kps["master"] = kubernetes.NewKubernetesFakeProvider("master", cps, dcs)
	router := mux.NewRouter()
	authenticator := auth.NewFakeAuthenticator(user)
	routing := NewRouting(ctx, dcs, kps, cps, authenticator, crdclientfake.NewSimpleClientset(), nil, nil)
	routing.Register(router)

	return router
}

func compareWithResult(t *testing.T, res *httptest.ResponseRecorder, file string) {
	bBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read response body")
	}

	rBytes, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatalf("Cannot read response file %s", file)
	}

	r := strings.TrimSpace(string(rBytes))
	b := strings.TrimSpace(string(bBytes))

	if r != b {
		t.Fatalf("Expected response body to be '%s', got '%s'", r, b)
	}
}

func encodeReq(t *testing.T, req interface{}) *bytes.Reader {
	b, err := json.Marshal(&req)
	if err != nil {
		t.Fatal(err)
	}

	return bytes.NewReader(b)
}

func getUser(admin bool) auth.User {
	u := auth.User{
		Name: "Thomas Tester",
		Roles: map[string]struct{}{
			"user": {},
		},
	}
	if admin {
		u.Roles["admin"] = struct{}{}
	}
	return u
}

func checkStatusCode(code int, recorder *httptest.ResponseRecorder, t *testing.T) {
	if recorder.Code != code {
		t.Errorf("Expected status code to be %d, got: %d", code, recorder.Code)
		t.Error(recorder.Body.String())
		return
	}
}

func TestUpRoute(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(false))
	e.ServeHTTP(res, req)
	checkStatusCode(http.StatusOK, res, t)
}
