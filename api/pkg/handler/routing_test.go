package handler

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticfakeclentset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	"k8s.io/apimachinery/pkg/runtime"
)

func createTestEndpoint(user apiv1.User, kubeObjects, kubermaticObjects []runtime.Object, versions map[string]*apiv1.MasterVersion, updates []apiv1.MasterUpdate,
) http.Handler {
	ctx := context.Background()

	datacenters := buildDatacenterMeta()
	cloudProviders := cloud.Providers(datacenters)

	authenticator := NewFakeAuthenticator(user)

	kubeClient := fake.NewSimpleClientset(kubeObjects...)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, 10*time.Millisecond)

	kubermaticClient := kubermaticfakeclentset.NewSimpleClientset(kubermaticObjects...)
	kubermaticInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticClient, 10*time.Millisecond)

	sshKeyProvider := kubernetes.NewSSHKeyProvider(kubermaticClient, kubermaticInformerFactory.Kubermatic().V1().UserSSHKeies().Lister(), IsAdmin)
	userProvider := kubernetes.NewUserProvider(kubermaticClient, kubermaticInformerFactory.Kubermatic().V1().Users().Lister())
	clusterProvider := kubernetes.NewClusterProvider(kubermaticClient, client.New(kubeInformerFactory.Core().V1().Secrets().Lister()), kubermaticInformerFactory.Kubermatic().V1().Clusters().Lister(), "", IsAdmin)
	clusterProviders := map[string]provider.ClusterProvider{"us-central1": clusterProvider}

	kubeInformerFactory.Start(wait.NeverStop)
	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)

	kubermaticInformerFactory.Start(wait.NeverStop)
	kubermaticInformerFactory.WaitForCacheSync(wait.NeverStop)

	r := NewRouting(
		ctx,
		datacenters,
		clusterProviders,
		cloudProviders,
		sshKeyProvider,
		userProvider,
		authenticator,
		versions,
		updates,
	)
	mainRouter := mux.NewRouter()
	v1Router := mainRouter.PathPrefix("/api/v1").Subrouter()
	v2Router := mainRouter.PathPrefix("/api/v2").Subrouter()
	v3Router := mainRouter.PathPrefix("/api/v3").Subrouter()
	r.RegisterV1(v1Router)
	r.RegisterV2(v2Router)
	r.RegisterV3(v3Router)

	return mainRouter
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

func getUser(name string, admin bool) apiv1.User {
	u := apiv1.User{
		ID: name,
		Roles: map[string]struct{}{
			"user": {},
		},
	}
	if admin {
		u.Roles[AdminRoleKey] = struct{}{}
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
	req := httptest.NewRequest("GET", "/api/v1/healthz", nil)
	res := httptest.NewRecorder()
	e := createTestEndpoint(getUser(testUsername, false), []runtime.Object{}, []runtime.Object{}, nil, nil)
	e.ServeHTTP(res, req)
	checkStatusCode(http.StatusOK, res, t)
}
