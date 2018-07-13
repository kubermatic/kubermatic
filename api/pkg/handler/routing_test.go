package handler

import (
	"encoding/json"
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
	"github.com/kubermatic/kubermatic/api/pkg/version"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"

	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
)

func createTestEndpointAndGetClients(user apiv1.User, dc map[string]provider.DatacenterMeta, kubeObjects, kubermaticObjects []runtime.Object, versions []*version.MasterVersion, updates []*version.MasterUpdate) (http.Handler, *kubermaticfakeclentset.Clientset, error) {

	datacenters := dc
	if datacenters == nil {
		datacenters = buildDatacenterMeta()
	}
	cloudProviders := cloud.Providers(datacenters)

	authenticator := NewFakeAuthenticator(user)

	kubeClient := fake.NewSimpleClientset(kubeObjects...)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, 10*time.Millisecond)

	kubermaticClient := kubermaticfakeclentset.NewSimpleClientset(kubermaticObjects...)
	kubermaticInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticClient, 10*time.Millisecond)

	fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (kubermaticclientv1.KubermaticV1Interface, error) {
		return kubermaticClient.KubermaticV1(), nil
	}

	sshKeyProvider := kubernetes.NewSSHKeyProvider(kubermaticClient, kubermaticInformerFactory.Kubermatic().V1().UserSSHKeies().Lister(), IsAdmin)
	newSSHKeyProvider := kubernetes.NewRBACCompliantSSHKeyProvider(fakeImpersonationClient, kubermaticInformerFactory.Kubermatic().V1().UserSSHKeies().Lister())
	userProvider := kubernetes.NewUserProvider(kubermaticClient, kubermaticInformerFactory.Kubermatic().V1().Users().Lister())
	projectProvider, err := kubernetes.NewProjectProvider(fakeImpersonationClient, kubermaticInformerFactory.Kubermatic().V1().Projects().Lister())
	if err != nil {
		return nil, nil, err
	}

	clusterProvider := kubernetes.NewClusterProvider(
		kubermaticClient,
		client.New(kubeInformerFactory.Core().V1().Secrets().Lister()),
		kubermaticInformerFactory.Kubermatic().V1().Clusters().Lister(),
		"",
		IsAdmin,
	)
	clusterProviders := map[string]provider.ClusterProvider{"us-central1": clusterProvider}

	newClusterProvider := kubernetes.NewRBACCompliantClusterProvider(
		fakeImpersonationClient,
		client.New(kubeInformerFactory.Core().V1().Secrets().Lister()),
		kubermaticInformerFactory.Kubermatic().V1().Clusters().Lister(),
		"",
	)
	newClusterProviders := map[string]provider.NewClusterProvider{"us-central1": newClusterProvider}

	kubeInformerFactory.Start(wait.NeverStop)
	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)

	kubermaticInformerFactory.Start(wait.NeverStop)
	kubermaticInformerFactory.WaitForCacheSync(wait.NeverStop)

	updateManager := version.New(versions, updates)

	// Disable the metrics endpoint in tests
	var promURL *string

	r := NewRouting(
		datacenters,
		clusterProviders,
		newClusterProviders,
		cloudProviders,
		sshKeyProvider,
		newSSHKeyProvider,
		userProvider,
		projectProvider,
		authenticator,
		updateManager,
		promURL,
	)
	mainRouter := mux.NewRouter()
	v1Router := mainRouter.PathPrefix("/api/v1").Subrouter()
	v3Router := mainRouter.PathPrefix("/api/v3").Subrouter()
	r.RegisterV1(v1Router)
	r.RegisterV3(v3Router)

	return mainRouter, kubermaticClient, nil
}

func createTestEndpoint(user apiv1.User, kubeObjects, kubermaticObjects []runtime.Object, versions []*version.MasterVersion, updates []*version.MasterUpdate) (http.Handler, error) {
	router, _, err := createTestEndpointAndGetClients(user, nil, kubeObjects, kubermaticObjects, versions, updates)
	return router, err
}

func createTestEndpointForDC(user apiv1.User, dc map[string]provider.DatacenterMeta, kubeObjects, kubermaticObjects []runtime.Object, versions []*version.MasterVersion, updates []*version.MasterUpdate) (http.Handler, error) {
	router, _, err := createTestEndpointAndGetClients(user, dc, kubeObjects, kubermaticObjects, versions, updates)
	return router, err
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
		"moon-1": {
			Location: "Dark Side",
			Seed:     "us-central1",
			Country:  "Moon States",
			Spec: provider.DatacenterSpec{
				VSphere: &provider.VSphereSpec{
					Endpoint:      "http://127.0.0.1:8989",
					AllowInsecure: true,
					Datastore:     "LocalDS_0",
					Datacenter:    "ha-datacenter",
					Cluster:       "localhost.localdomain",
					RootPath:      "/ha-datacenter/vm/",
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

func compareJSON(t *testing.T, res *httptest.ResponseRecorder, s2 string) {
	var o1 interface{}
	var o2 interface{}

	// var err error
	bBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read response body")
	}
	err = json.Unmarshal(bBytes, &o1)
	if err != nil {
		t.Fatalf("Error marshaling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		t.Fatalf("Error marshaling string 2 :: %s", err.Error())
	}
	if !equality.Semantic.DeepEqual(o1, o2) {
		t.Fatalf("Objects are different: %v", diff.ObjectDiff(o1, o2))
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
	ep, err := createTestEndpoint(getUser(testUsername, false), []runtime.Object{}, []runtime.Object{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)
	checkStatusCode(http.StatusOK, res, t)
}
