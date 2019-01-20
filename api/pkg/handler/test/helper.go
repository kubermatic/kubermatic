package test

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusapi "github.com/prometheus/client_golang/api"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticfakeclentset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	kubernetesclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"

	clusterclientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
	fakeclusterclientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/fake"
)

const (
	// UserID holds the test user ID
	UserID = "1233"
	// UserName holds the test user name
	UserName = "user1"
	// UserEmail holds the test user email
	UserEmail = "john@acme.com"
	// ClusterID holds the test cluster ID
	ClusterID = "AbcClusterID"
	// DefaultClusterID holds the test default cluster ID
	DefaultClusterID = "defClusterID"
	// DefaultClusterName holds the test default cluster name
	DefaultClusterName = "defClusterName"
	// ProjectName holds the test project ID
	ProjectName = "my-first-project-ID"
)

// GetUser is a convenience function for generating apiv1.User
func GetUser(email, id, name string, admin bool) apiv1.User {
	u := apiv1.User{
		ObjectMeta: apiv1.ObjectMeta{
			ID:   id,
			Name: name,
		},
		Email: email,
	}
	return u
}

// newRoutingFunc defines a func that knows how to create and set up routing required for testing
// this function is temporal until all types end up in their own packages.
// it is meant to be used by legacy handler.createTestEndpointAndGetClients function
type newRoutingFunc func(
	datacenters map[string]provider.DatacenterMeta,
	newClusterProviders map[string]provider.ClusterProvider,
	cloudProviders map[string]provider.CloudProvider,
	newSSHKeyProvider provider.SSHKeyProvider,
	userProvider provider.UserProvider,
	projectProvider provider.ProjectProvider,
	oidcAuthenticator auth.OIDCAuthenticator,
	oidcIssuerVerifier auth.OIDCIssuerVerifier,
	prometheusClient prometheusapi.Client,
	projectMemberProvider *kubernetes.ProjectMemberProvider,
	versions []*version.MasterVersion,
	updates []*version.MasterUpdate) http.Handler

// CreateTestEndpointAndGetClients is a convenience function that instantiates fake providers and sets up routes  for the tests
func CreateTestEndpointAndGetClients(user apiv1.User, dc map[string]provider.DatacenterMeta, kubeObjects, machineObjects, kubermaticObjects []runtime.Object, versions []*version.MasterVersion, updates []*version.MasterUpdate, routingFunc newRoutingFunc) (http.Handler, *ClientsSets, error) {
	datacenters := dc
	if datacenters == nil {
		datacenters = buildDatacenterMeta()
	}
	cloudProviders := cloud.Providers(datacenters)
	authenticator := NewAuthenticator(user)
	issuerVerifier := NewIssuerVerifier()

	kubeClient := fake.NewSimpleClientset(kubeObjects...)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, 10*time.Millisecond)
	kubermaticClient := kubermaticfakeclentset.NewSimpleClientset(kubermaticObjects...)
	kubermaticInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticClient, 10*time.Millisecond)

	fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (kubermaticclientv1.KubermaticV1Interface, error) {
		return kubermaticClient.KubermaticV1(), nil
	}

	sshKeyProvider := kubernetes.NewSSHKeyProvider(fakeImpersonationClient, kubermaticInformerFactory.Kubermatic().V1().UserSSHKeys().Lister())
	userProvider := kubernetes.NewUserProvider(kubermaticClient, kubermaticInformerFactory.Kubermatic().V1().Users().Lister())
	projectMemberProvider := kubernetes.NewProjectMemberProvider(fakeImpersonationClient, kubermaticInformerFactory.Kubermatic().V1().UserProjectBindings().Lister())
	projectProvider, err := kubernetes.NewProjectProvider(fakeImpersonationClient, kubermaticInformerFactory.Kubermatic().V1().Projects().Lister())
	if err != nil {
		return nil, nil, err
	}

	fakeMachineClient := fakeclusterclientset.NewSimpleClientset(machineObjects...)
	fUserClusterConnection := &fakeUserClusterConnection{fakeMachineClient, kubeClient}

	clusterProvider := kubernetes.NewClusterProvider(
		fakeImpersonationClient,
		fUserClusterConnection,
		kubermaticInformerFactory.Kubermatic().V1().Clusters().Lister(),
		"",
	)
	clusterProviders := map[string]provider.ClusterProvider{"us-central1": clusterProvider}

	kubeInformerFactory.Start(wait.NeverStop)
	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)
	kubermaticInformerFactory.Start(wait.NeverStop)
	kubermaticInformerFactory.WaitForCacheSync(wait.NeverStop)

	// Disable the metrics endpoint in tests
	var prometheusClient prometheusapi.Client

	mainRouter := routingFunc(
		datacenters,
		clusterProviders,
		cloudProviders,
		sshKeyProvider,
		userProvider,
		projectProvider,
		authenticator,
		issuerVerifier,
		prometheusClient,
		projectMemberProvider,
		versions,
		updates,
	)

	return mainRouter, &ClientsSets{kubermaticClient, fakeMachineClient}, nil
}

// CreateTestEndpoint does exactly the same as CreateTestEndpointAndGetClients except it omits ClientsSets when returning
func CreateTestEndpoint(user apiv1.User, kubeObjects, kubermaticObjects []runtime.Object, versions []*version.MasterVersion, updates []*version.MasterUpdate, routingFunc newRoutingFunc) (http.Handler, error) {
	router, _, err := CreateTestEndpointAndGetClients(user, nil, kubeObjects, nil, kubermaticObjects, versions, updates, routingFunc)
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

type fakeUserClusterConnection struct {
	fakeMachineClient    clusterclientset.Interface
	fakeKubernetesClient kubernetesclient.Interface
}

func (f *fakeUserClusterConnection) GetAdminKubeconfig(c *kubermaticapiv1.Cluster) ([]byte, error) {
	return []byte(generateTestKubeconfig(ClusterID, IDToken)), nil
}

func (f *fakeUserClusterConnection) GetMachineClient(c *kubermaticapiv1.Cluster) (clusterclientset.Interface, error) {
	return f.fakeMachineClient, nil
}

func (f *fakeUserClusterConnection) GetClient(c *kubermaticapiv1.Cluster) (kubernetesclient.Interface, error) {
	return f.fakeKubernetesClient, nil
}

// ClientsSets a simple wrapper that holds fake client sets
type ClientsSets struct {
	FakeKubermaticClient *kubermaticfakeclentset.Clientset
	FakeMachineClient    *fakeclusterclientset.Clientset
}

// generateTestKubeconfig returns test kubeconfig yaml structure
func generateTestKubeconfig(clusterID, token string) string {
	return fmt.Sprintf(`
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: 
    server: test.fake.io
  name: %s
contexts:
- context:
    cluster: %s
    user: default
  name: default
current-context: default
kind: Config
users:
- name: default
  user:
    token: %s`, clusterID, clusterID, token)
}

// APIUserToKubermaticUser simply converts apiv1.User to kubermaticapiv1.User type
func APIUserToKubermaticUser(user apiv1.User) *kubermaticapiv1.User {
	return &kubermaticapiv1.User{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: kubermaticapiv1.UserSpec{
			Name:  user.Name,
			Email: user.Email,
			ID:    user.ID,
		},
	}
}

// CompareWithResult a convenience function for comparing http.Body content with response
func CompareWithResult(t *testing.T, res *httptest.ResponseRecorder, response string) {
	t.Helper()
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

// GenUser generates a User resource
// note if the id is empty then it will be auto generated
func GenUser(id, name, email string) *kubermaticapiv1.User {
	if len(id) == 0 {
		// the name of the object is derived from the email address and encoded as sha256
		id = fmt.Sprintf("%x", sha256.Sum256([]byte(email)))
	}
	return &kubermaticapiv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
			UID:  types.UID(fmt.Sprintf("fake-uid-%s", id)),
		},
		Spec: kubermaticapiv1.UserSpec{
			Name:  name,
			Email: email,
		},
	}
}

// GenAPIUser generates a API user
func GenAPIUser(name, email string) *apiv1.User {
	usr := GenUser("", name, email)
	return &apiv1.User{
		ObjectMeta: apiv1.ObjectMeta{
			ID:   usr.Name,
			Name: usr.Spec.Name,
		},
		Email: usr.Spec.Email,
	}
}

// DefaultCreationTimestamp returns default test timestamp
func DefaultCreationTimestamp() time.Time {
	return time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)
}

// GenDefaultAPIUser generates a default API user
func GenDefaultAPIUser() *apiv1.User {
	return &apiv1.User{
		ObjectMeta: apiv1.ObjectMeta{
			ID:   GenDefaultUser().Name,
			Name: GenDefaultUser().Spec.Name,
		},
		Email: GenDefaultUser().Spec.Email,
	}
}

// GenDefaultUser generates a default user
func GenDefaultUser() *kubermaticapiv1.User {
	userEmail := "bob@acme.com"
	return GenUser("", "Bob", userEmail)
}

// GenProject generates new empty project
func GenProject(name, phase string, creationTime time.Time, oRef ...metav1.OwnerReference) *kubermaticapiv1.Project {
	return &kubermaticapiv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:              fmt.Sprintf("%s-%s", name, "ID"),
			CreationTimestamp: metav1.NewTime(creationTime),
			OwnerReferences:   oRef,
		},
		Spec: kubermaticapiv1.ProjectSpec{Name: name},
		Status: kubermaticapiv1.ProjectStatus{
			Phase: phase,
		},
	}
}

// GenDefaultProject generates a default project
func GenDefaultProject() *kubermaticapiv1.Project {
	user := GenDefaultUser()
	oRef := metav1.OwnerReference{
		APIVersion: "kubermatic.io/v1",
		Kind:       "User",
		UID:        user.UID,
		Name:       user.Name,
	}
	return GenProject("my-first-project", kubermaticapiv1.ProjectActive, DefaultCreationTimestamp(), oRef)
}

// GenBinding generates a binding
func GenBinding(projectID, email, group string) *kubermaticapiv1.UserProjectBinding {
	return &kubermaticapiv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s-%s", projectID, email, group),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
					Kind:       kubermaticapiv1.ProjectKindName,
					Name:       projectID,
				},
			},
		},
		Spec: kubermaticapiv1.UserProjectBindingSpec{
			UserEmail: email,
			ProjectID: projectID,
			Group:     fmt.Sprintf("%s-%s", group, projectID),
		},
	}
}

// GenDefaultOwnerBinding generates default owner binding
func GenDefaultOwnerBinding() *kubermaticapiv1.UserProjectBinding {
	return GenBinding(GenDefaultProject().Name, GenDefaultUser().Spec.Email, "owners")
}

// GenDefaultKubermaticObjects generates default kubermatic object
func GenDefaultKubermaticObjects(objs ...runtime.Object) []runtime.Object {
	defaultsObjs := []runtime.Object{
		// add a project
		GenDefaultProject(),
		// add a user
		GenDefaultUser(),
		// make a user the owner of the default project
		GenDefaultOwnerBinding(),
	}

	return append(defaultsObjs, objs...)
}

func GenCluster(id string, name string, projectID string, creationTime time.Time) *kubermaticapiv1.Cluster {
	return &kubermaticapiv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   id,
			Labels: map[string]string{"project-id": projectID},
			CreationTimestamp: func() metav1.Time {
				return metav1.NewTime(creationTime)
			}(),
		},
		Spec: kubermaticapiv1.ClusterSpec{
			Cloud: kubermaticapiv1.CloudSpec{
				DatacenterName: "FakeDatacenter",
				Fake:           &kubermaticapiv1.FakeCloudSpec{Token: "SecretToken"},
			},
			Version:           *semver.NewSemverOrDie("9.9.9"),
			HumanReadableName: name,
		},
		Address: kubermaticapiv1.ClusterAddress{
			AdminToken:   "drphc2.g4kq82pnlfqjqt65",
			ExternalName: "w225mx4z66.asia-east1-a-1.cloud.kubermatic.io",
			IP:           "35.194.142.199",
			URL:          "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
		},
		Status: kubermaticapiv1.ClusterStatus{
			Health: kubermaticapiv1.ClusterHealth{
				ClusterHealthStatus: kubermaticapiv1.ClusterHealthStatus{
					Apiserver:         true,
					Scheduler:         true,
					Controller:        true,
					MachineController: true,
					Etcd:              true,
				},
			},
		},
	}
}

func GenDefaultCluster() *kubermaticapiv1.Cluster {
	return GenCluster(DefaultClusterID, DefaultClusterName, GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
}

func CheckStatusCode(wantStatusCode int, recorder *httptest.ResponseRecorder, t *testing.T) {
	t.Helper()
	if recorder.Code != wantStatusCode {
		t.Errorf("Expected status code to be %d, got: %d", wantStatusCode, recorder.Code)
		t.Error(recorder.Body.String())
		return
	}
}
