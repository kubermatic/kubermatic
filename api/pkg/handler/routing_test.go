package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticfakeclentset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/version"
	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"
	fakemachineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned/fake"

	prometheusapi "github.com/prometheus/client_golang/api"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	kubernetesclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
)

func createTestEndpointAndGetClients(user apiv1.User, dc map[string]provider.DatacenterMeta, kubeObjects, machineObjects, kubermaticObjects []runtime.Object, versions []*version.MasterVersion, updates []*version.MasterUpdate) (http.Handler, *clientsSets, error) {
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
	projectMemberProvider := kubernetes.NewProjectMemberProvider(fakeImpersonationClient, kubermaticInformerFactory.Kubermatic().V1().UserProjectBindings().Lister())
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
	fakeMachineClient := fakemachineclientset.NewSimpleClientset(machineObjects...)
	fUserClusterConnection := &fakeUserClusterConnection{fakeMachineClient, kubeClient}

	newClusterProvider := kubernetes.NewRBACCompliantClusterProvider(
		fakeImpersonationClient,
		fUserClusterConnection,
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
	var prometheusClient prometheusapi.Client

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
		prometheusClient,
		projectMemberProvider,
		projectMemberProvider, /*satisfies also a different interface*/
	)
	mainRouter := mux.NewRouter()
	v1Router := mainRouter.PathPrefix("/api/v1").Subrouter()
	v3Router := mainRouter.PathPrefix("/api/v3").Subrouter()
	r.RegisterV1(v1Router)
	r.RegisterV3(v3Router)

	return mainRouter, &clientsSets{kubermaticClient, fakeMachineClient}, nil
}

func createTestEndpoint(user apiv1.User, kubeObjects, kubermaticObjects []runtime.Object, versions []*version.MasterVersion, updates []*version.MasterUpdate) (http.Handler, error) {
	router, _, err := createTestEndpointAndGetClients(user, nil, kubeObjects, nil, kubermaticObjects, versions, updates)
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
	}
}

func deepEqualUnorderedList(a []interface{}, b []interface{}) bool {

	if len(a) != len(b) {
		return false
	}

	matched := make([]bool, len(b))

	for _, v1 := range a {
		foundMatch := false
		for j, v2 := range b {
			if matched[j] {
				continue
			}
			if equality.Semantic.DeepEqual(v1, v2) {
				foundMatch = true
				matched[j] = true
				break
			}
		}
		if !foundMatch {
			return false
		}
	}

	return true
}

// use this function if the order in the JSON list doesn't matter: `{[ "Foo", "Bar" ]}` == `{[ "Bar", "Foo" ]}`
func compareUnorderedJSON(t *testing.T, res *httptest.ResponseRecorder, expectedResponseString string) {
	t.Helper()
	var actualResponse interface{}
	var expectedResponse interface{}

	// var err error
	bBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read response body")
	}
	err = json.Unmarshal(bBytes, &actualResponse)
	if err != nil {
		t.Fatalf("Error marshaling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(expectedResponseString), &expectedResponse)
	if err != nil {
		t.Fatalf("Error marshaling string 2 :: %s", err.Error())
	}

	switch actualResponse.(type) {
	case []interface{}:
		if !deepEqualUnorderedList(actualResponse.([]interface{}), expectedResponse.([]interface{})) {
			t.Fatalf("Objects are different: %v", diff.ObjectDiff(actualResponse, expectedResponse))
		}
	default:
		if !equality.Semantic.DeepEqual(actualResponse, expectedResponse) {
			t.Fatalf("Objects are different: %v", diff.ObjectDiff(actualResponse, expectedResponse))
		}
	}
}

func compareJSON(t *testing.T, res *httptest.ResponseRecorder, expectedResponseString string) {
	t.Helper()
	var actualResponse interface{}
	var expectedResponse interface{}

	// var err error
	bBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read response body")
	}
	err = json.Unmarshal(bBytes, &actualResponse)
	if err != nil {
		t.Fatalf("Error marshaling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(expectedResponseString), &expectedResponse)
	if err != nil {
		t.Fatalf("Error marshaling string 2 :: %s", err.Error())
	}
	if !equality.Semantic.DeepEqual(actualResponse, expectedResponse) {
		t.Fatalf("Objects are different: %v", diff.ObjectDiff(actualResponse, expectedResponse))
	}
}

// areEqualOrDie checks if binary representation of actual and expected is equal.
//
// note that:
// this function fails when conversion is not possible
func areEqualOrDie(t *testing.T, actual, expected interface{}) bool {
	t.Helper()
	actualBytes, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("failed to marshal actual: %v", err)
	}

	expectedBytes, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("failed to marshal expected: %v", err)
	}
	return bytes.Equal(actualBytes, expectedBytes)
}

const (
	testUserID    = "1233"
	testUserName  = "user1"
	testUserEmail = "john@acme.com"
)

func getUser(email, id, name string, admin bool) apiv1.User {
	u := apiv1.User{
		ID:    id,
		Name:  name,
		Email: email,
		Roles: map[string]struct{}{
			"user": {},
		},
	}
	if admin {
		u.Roles[AdminRoleKey] = struct{}{}
	}
	return u
}

func apiUserToKubermaticUser(user apiv1.User) *kubermaticapiv1.User {
	return &kubermaticapiv1.User{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: kubermaticapiv1.UserSpec{
			Name:  user.Name,
			Email: user.Email,
			ID:    user.ID,
		},
	}
}

func checkStatusCode(wantStatusCode int, recorder *httptest.ResponseRecorder, t *testing.T) {
	t.Helper()
	if recorder.Code != wantStatusCode {
		t.Errorf("Expected status code to be %d, got: %d", wantStatusCode, recorder.Code)
		t.Error(recorder.Body.String())
		return
	}
}

func TestUpRoute(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v1/healthz", nil)
	res := httptest.NewRecorder()
	ep, err := createTestEndpoint(getUser(testUserEmail, testUserID, testUserName, false), []runtime.Object{}, []runtime.Object{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)
	checkStatusCode(http.StatusOK, res, t)
}

type fakeUserClusterConnection struct {
	fakeMachineClient    machineclientset.Interface
	fakeKubernetesClient kubernetesclient.Interface
}

func (f *fakeUserClusterConnection) GetAdminKubeconfig(c *kubermaticapiv1.Cluster) ([]byte, error) {
	return []byte{}, errors.New("not yet implemented")
}

func (f *fakeUserClusterConnection) GetMachineClient(c *kubermaticapiv1.Cluster) (machineclientset.Interface, error) {
	return f.fakeMachineClient, nil
}

func (f *fakeUserClusterConnection) GetClient(c *kubermaticapiv1.Cluster) (kubernetesclient.Interface, error) {
	return f.fakeKubernetesClient, nil
}

type clientsSets struct {
	fakeKubermaticClient *kubermaticfakeclentset.Clientset
	fakeMachineClient    *fakemachineclientset.Clientset
}
