package handler

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/kubermatic/kubermatic/api"
	mastercrdfake "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned/fake"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubermatic"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"k8s.io/apimachinery/pkg/runtime"
)

func createTestEndpoint(user auth.User, masterCrdObjects []runtime.Object, versions map[string]*api.MasterVersion, updates []api.MasterUpdate,
) http.Handler {
	ctx := context.Background()

	dcs := buildDatacenterMeta()
	// create CloudProviders
	cps := cloud.Providers(dcs)
	router := mux.NewRouter()
	authenticator := auth.NewFakeAuthenticator(user)
	masterCrdClient := mastercrdfake.NewSimpleClientset(masterCrdObjects...)
	kp := kubernetes.NewKubernetesProvider(masterCrdClient, cps, "", dcs)
	dataProvider := kubermatic.New(masterCrdClient)

	routing := NewRouting(ctx, dcs, kp, cps, authenticator, dataProvider, versions, updates)
	routing.Register(router)

	return router
}

func buildDatacenterMeta() map[string]provider.DatacenterMeta {
	return map[string]provider.DatacenterMeta{
		"us-central1": {
			Location: "us-central",
			Country:  "US",
			Private:  false,
			IsSeed:   true,
			Spec: provider.DatacenterSpec{
				Digitalocean: &provider.DigitaloceanSpec{
					Region: "ams2",
				},
			},
		},
		"private-do1": {
			Location: "US ",
			Seed:     "us-central1",
			Country:  "NL",
			Private:  true,
			Spec: provider.DatacenterSpec{
				Digitalocean: &provider.DigitaloceanSpec{
					Region: "ams2",
				},
			},
		},
		"regular-do1": {
			Location: "Amsterdam",
			Seed:     "us-central1",
			Country:  "NL",
			Spec: provider.DatacenterSpec{
				Digitalocean: &provider.DigitaloceanSpec{
					Region: "ams2",
				},
			},
		},
	}
}

func compareWithResult(t *testing.T, res *httptest.ResponseRecorder, response string) {
	bBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read response body")
	}

	r := strings.TrimSpace(response)
	b := strings.TrimSpace(string(bBytes))

	if r != b {
		t.Fatalf("Expected response body to be \n%s \ngot \n%s", r, b)
	}
}

const (
	testUsername = "user1"
)

func getUser(name string, admin bool) auth.User {
	u := auth.User{
		Name: name,
		Roles: map[string]struct{}{
			"user": {},
		},
	}
	if admin {
		u.Roles["admin"] = struct{}{}
	}
	return u
}

func checkStatusCode(wantStatusCode int, recorder *httptest.ResponseRecorder, t *testing.T) {
	if recorder.Code != wantStatusCode {
		t.Errorf("Expected status code to be %d, got: %d", wantStatusCode, recorder.Code)
		t.Error(recorder.Body.String())
		return
	}
}

func TestUpRoute(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(testUsername, false), []runtime.Object{}, nil, nil)
	e.ServeHTTP(res, req)
	checkStatusCode(http.StatusOK, res, t)
}
