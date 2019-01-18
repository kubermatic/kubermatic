package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/go-test/deep"
	"github.com/gorilla/mux"
	prometheusapi "github.com/prometheus/client_golang/api"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticfakeclentset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	"k8s.io/apimachinery/pkg/runtime"
	fakeclusterclientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/fake"
)

// newTestRouting defines a func that knows how to create and set up routing required for testing
// this function is temporal until all types end up in their own packages.
// it also helps us avoid circular imports
// for example handler package uses test pkg that needs handler for setting up the routing (NewRouting function)
func newTestRouting(
	datacenters map[string]provider.DatacenterMeta,
	clusterProviders map[string]provider.ClusterProvider,
	cloudProviders map[string]provider.CloudProvider,
	sshKeyProvider provider.SSHKeyProvider,
	userProvider provider.UserProvider,
	projectProvider provider.ProjectProvider,
	authenticator auth.OIDCAuthenticator,
	issuerVerifier auth.OIDCIssuerVerifier,
	prometheusClient prometheusapi.Client,
	projectMemberProvider *kubernetes.ProjectMemberProvider,
	versions []*version.MasterVersion,
	updates []*version.MasterUpdate) http.Handler {

	updateManager := version.New(versions, updates)
	r := NewRouting(
		datacenters,
		clusterProviders,
		cloudProviders,
		sshKeyProvider,
		userProvider,
		projectProvider,
		authenticator,
		issuerVerifier,
		updateManager,
		prometheusClient,
		projectMemberProvider,
		projectMemberProvider, /*satisfies also a different interface*/
	)

	mainRouter := mux.NewRouter()
	v1Router := mainRouter.PathPrefix("/api/v1").Subrouter()
	r.RegisterV1(v1Router)
	r.RegisterV1Optional(v1Router,
		true,
		*generateDefaultOicdCfg(),
		mainRouter,
	)
	return mainRouter
}

func createTestEndpointAndGetClients(user apiv1.User, dc map[string]provider.DatacenterMeta, kubeObjects, machineObjects, kubermaticObjects []runtime.Object, versions []*version.MasterVersion, updates []*version.MasterUpdate) (http.Handler, *clientsSets, error) {
	handler, cs, err := test.CreateTestEndpointAndGetClients(user, dc, kubeObjects, machineObjects, kubermaticObjects, versions, updates, newTestRouting)
	if err != nil {
		return nil, nil, err
	}
	return handler, &clientsSets{cs.FakeKubermaticClient, cs.FakeMachineClient}, nil
}

func createTestEndpoint(user apiv1.User, kubeObjects, kubermaticObjects []runtime.Object, versions []*version.MasterVersion, updates []*version.MasterUpdate) (http.Handler, error) {
	router, _, err := createTestEndpointAndGetClients(user, nil, kubeObjects, nil, kubermaticObjects, versions, updates)
	return router, err
}

func compareWithResult(t *testing.T, res *httptest.ResponseRecorder, response string) {
	test.CompareWithResult(t, res, response)
}

const (
	testUserID    = test.UserID
	testUserName  = test.UserName
	testUserEmail = test.UserEmail
)

func getUser(email, id, name string, admin bool) apiv1.User {
	return test.GetUser(email, id, name, admin)
}

func apiUserToKubermaticUser(user apiv1.User) *kubermaticapiv1.User {
	return test.APIUserToKubermaticUser(user)
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
	test.CheckStatusCode(http.StatusOK, res, t)
}

type clientsSets struct {
	fakeKubermaticClient *kubermaticfakeclentset.Clientset
	fakeMachineClient    *fakeclusterclientset.Clientset
}

// new>SSHKeyV1SliceWrapper wraps []apiv1.SSHKey
// to provide convenient methods for tests
type newSSHKeyV1SliceWrapper []apiv1.SSHKey

// Sort sorts the collection by CreationTimestamp
func (k newSSHKeyV1SliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].CreationTimestamp.Before(k[j].CreationTimestamp)
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *newSSHKeyV1SliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *newSSHKeyV1SliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k newSSHKeyV1SliceWrapper) EqualOrDie(expected newSSHKeyV1SliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// newUserV1SliceWrapper wraps []apiv1.User
// to provide convenient methods for tests
type newUserV1SliceWrapper []apiv1.User

// Sort sorts the collection by CreationTimestamp
func (k newUserV1SliceWrapper) Sort() {
	sort.Slice(k, func(i, j int) bool {
		return k[i].CreationTimestamp.Before(k[j].CreationTimestamp)
	})
}

// DecodeOrDie reads and decodes json data from the reader
func (k *newUserV1SliceWrapper) DecodeOrDie(r io.Reader, t *testing.T) *newUserV1SliceWrapper {
	t.Helper()
	dec := json.NewDecoder(r)
	err := dec.Decode(k)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// EqualOrDie compares whether expected collection is equal to the actual one
func (k newUserV1SliceWrapper) EqualOrDie(expected newUserV1SliceWrapper, t *testing.T) {
	t.Helper()
	if diff := deep.Equal(k, expected); diff != nil {
		t.Errorf("actual slice is different that the expected one. Diff: %v", diff)
	}
}

// generateDefaultOicdCfg creates test configuration for OpenID clients
func generateDefaultOicdCfg() *OIDCConfiguration {
	return &OIDCConfiguration{
		URL:                  test.IssuerURL,
		ClientID:             test.IssuerClientID,
		ClientSecret:         test.IssuerClientSecret,
		OfflineAccessAsScope: true,
	}
}
