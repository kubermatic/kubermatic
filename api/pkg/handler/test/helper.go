package test

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	prometheusapi "github.com/prometheus/client_golang/api"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	k8cuserclusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	kubermaticfakeclentset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	kubermaticinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/presets"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"
	"github.com/kubermatic/kubermatic/api/pkg/version"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	kubernetesclientset "k8s.io/client-go/kubernetes"
	fakerestclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	// We call this in init because even thought it is possible to register the same
	// scheme multiple times it is an unprotected concurrent map access and these tests
	// are very good at making that panic
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to add clusterv1alpha1 scheme to scheme.Scheme", "error", err)
	}
	if err := v1beta1.AddToScheme(scheme.Scheme); err != nil {
		kubermaticlog.Logger.Fatalw("failed to register scheme v1beta1", "error", err)
	}
}

const (
	// UserID holds a test user ID
	UserID = "1233"
	// UserID2 holds a test user ID
	UserID2 = "1523"
	// UserName holds a test user name
	UserName = "user1"
	// UserName2 holds a test user name
	UserName2 = "user2"
	// UserEmail holds a test user email
	UserEmail = "john@acme.com"
	// UserEmail2 holds a test user email
	UserEmail2 = "bob@example.com"
	// ClusterID holds the test cluster ID
	ClusterID = "AbcClusterID"
	// DefaultClusterID holds the test default cluster ID
	DefaultClusterID = "defClusterID"
	// DefaultClusterName holds the test default cluster name
	DefaultClusterName = "defClusterName"
	// ProjectName holds the test project ID
	ProjectName = "my-first-project-ID"
	// TestDatacenter holds datacenter name
	TestSeedDatacenter = "us-central1"
	// TestServiceAccountHashKey authenticates the service account's token value using HMAC
	TestServiceAccountHashKey = "eyJhbGciOiJIUzI1NeyJhbGciOiJIUzI1N"
	// TestFakeToken signed JWT token with fake data
	TestFakeToken = "eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6IjEiLCJleHAiOjE2NDk3NDg4NTYsImlhdCI6MTU1NTA1NDQ1NiwibmJmIjoxNTU1MDU0NDU2LCJwcm9qZWN0X2lkIjoiMSIsInRva2VuX2lkIjoiMSJ9.Q4qxzOaCvUnWfXneY654YiQjUTd_Lsmw56rE17W2ouo"
	// TestOSdomain OpenStack domain
	TestOSdomain = "OSdomain"
	// TestOSuserPass OpenStack user password
	TestOSuserPass = "OSpass"
	// TestOSuserName OpenStack user name
	TestOSuserName = "OSuser"
	// TestFakeCredential Fake provider credential name
	TestFakeCredential = "fake"
	// RequiredEmailDomain required domain for predefined credentials
	RequiredEmailDomain = "acme.com"
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
	settingsProvider provider.SettingsProvider,
	userInfoGetter provider.UserInfoGetter,
	seedsGetter provider.SeedsGetter,
	clusterProviderGetter provider.ClusterProviderGetter,
	addonProviderGetter provider.AddonProviderGetter,
	newSSHKeyProvider provider.SSHKeyProvider,
	userProvider provider.UserProvider,
	serviceAccountProvider provider.ServiceAccountProvider,
	serviceAccountTokenProvider provider.ServiceAccountTokenProvider,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	oidcIssuerVerifier auth.OIDCIssuerVerifier,
	tokenVerifiers auth.TokenVerifier,
	tokenExtractors auth.TokenExtractor,
	prometheusClient prometheusapi.Client,
	projectMemberProvider *kubernetes.ProjectMemberProvider,
	versions []*version.Version,
	updates []*version.Update,
	saTokenAuthenticator serviceaccount.TokenAuthenticator,
	saTokenGenerator serviceaccount.TokenGenerator,
	eventRecorderProvider provider.EventRecorderProvider,
	credentialManager common.PresetsManager) http.Handler

func initTestEndpoint(user apiv1.User, seedsGetter provider.SeedsGetter, kubeObjects, machineObjects, kubermaticObjects []runtime.Object, versions []*version.Version, updates []*version.Update, credentialsManager common.PresetsManager, routingFunc newRoutingFunc) (http.Handler, *ClientsSets, error) {
	if seedsGetter == nil {
		seedsGetter = buildSeeds()
	}
	allObjects := kubeObjects
	allObjects = append(allObjects, machineObjects...)
	allObjects = append(allObjects, kubermaticObjects...)
	fakeClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, allObjects...)
	kubermaticClient := kubermaticfakeclentset.NewSimpleClientset(kubermaticObjects...)
	kubermaticInformerFactory := kubermaticinformers.NewSharedInformerFactory(kubermaticClient, 10*time.Millisecond)
	kubernetesClient := fakerestclient.NewSimpleClientset(kubeObjects...)
	kubernetesInformerFactory := informers.NewSharedInformerFactory(kubernetesClient, 10*time.Millisecond)
	fakeKubermaticImpersonationClient := func(impCfg restclient.ImpersonationConfig) (kubermaticclientv1.KubermaticV1Interface, error) {
		return kubermaticClient.KubermaticV1(), nil
	}
	fakeKubernetesImpersonationClient := func(impCfg restclient.ImpersonationConfig) (kubernetesclientset.Interface, error) {
		return kubernetesClient, nil
	}

	userLister := kubermaticInformerFactory.Kubermatic().V1().Users().Lister()
	sshKeyProvider := kubernetes.NewSSHKeyProvider(fakeKubermaticImpersonationClient, kubermaticInformerFactory.Kubermatic().V1().UserSSHKeys().Lister())
	userProvider := kubernetes.NewUserProvider(kubermaticClient, userLister, kubernetes.IsServiceAccount)
	settingsProvider := kubernetes.NewSettingsProvider(kubermaticClient, kubermaticInformerFactory.Kubermatic().V1().KubermaticSettings().Lister())
	tokenGenerator, err := serviceaccount.JWTTokenGenerator([]byte(TestServiceAccountHashKey))
	if err != nil {
		return nil, nil, err
	}
	tokenAuth := serviceaccount.JWTTokenAuthenticator([]byte(TestServiceAccountHashKey))
	serviceAccountTokenProvider, err := kubernetes.NewServiceAccountTokenProvider(fakeKubernetesImpersonationClient, kubernetesInformerFactory.Core().V1().Secrets().Lister())
	if err != nil {
		return nil, nil, err
	}
	serviceAccountProvider := kubernetes.NewServiceAccountProvider(fakeKubermaticImpersonationClient, userLister, "localhost")
	projectMemberProvider := kubernetes.NewProjectMemberProvider(fakeKubermaticImpersonationClient, kubermaticInformerFactory.Kubermatic().V1().UserProjectBindings().Lister(), userLister, kubernetes.IsServiceAccount)
	userInfoGetter, err := provider.UserInfoGetterFactory(projectMemberProvider)
	if err != nil {
		return nil, nil, err
	}
	verifiers := []auth.TokenVerifier{}
	extractors := []auth.TokenExtractor{}
	{
		// if the API users is actually a service account use JWTTokenAuthentication
		// that knows how to extract and verify the token
		if strings.HasPrefix(user.Email, "serviceaccount-") {
			saExtractorVerifier := auth.NewServiceAccountAuthClient(
				auth.NewHeaderBearerTokenExtractor("Authorization"),
				serviceaccount.JWTTokenAuthenticator([]byte(TestServiceAccountHashKey)),
				serviceAccountTokenProvider,
			)
			verifiers = append(verifiers, saExtractorVerifier)
			extractors = append(extractors, saExtractorVerifier)

			// for normal users we use OIDCClient which is broken at the moment
			// because the tests don't send a token in the Header instead
			// the client spits out a hardcoded value
		} else {
			fakeOIDCClient := NewFakeOIDCClient(user)
			verifiers = append(verifiers, fakeOIDCClient)
			extractors = append(extractors, fakeOIDCClient)
		}
	}
	tokenVerifiers := auth.NewTokenVerifierPlugins(verifiers)
	tokenExtractors := auth.NewTokenExtractorPlugins(extractors)
	fakeOIDCClient := NewFakeOIDCClient(user)

	projectProvider, err := kubernetes.NewProjectProvider(fakeKubermaticImpersonationClient, kubermaticInformerFactory.Kubermatic().V1().Projects().Lister())
	if err != nil {
		return nil, nil, err
	}
	privilegedProjectProvider, err := kubernetes.NewPrivilegedProjectProvider(fakeKubermaticImpersonationClient)
	if err != nil {
		return nil, nil, err
	}

	fUserClusterConnection := &fakeUserClusterConnection{fakeClient}
	clusterProvider := kubernetes.NewClusterProvider(
		&restclient.Config{},
		fakeKubermaticImpersonationClient,
		fUserClusterConnection,
		"",
		rbac.ExtractGroupPrefix,
		fakeClient,
		kubernetesClient,
		false,
	)
	clusterProviders := map[string]provider.ClusterProvider{"us-central1": clusterProvider}
	clusterProviderGetter := func(seed *kubermaticv1.Seed) (provider.ClusterProvider, error) {
		if clusterProvider, exists := clusterProviders[seed.Name]; exists {
			return clusterProvider, nil
		}
		return nil, fmt.Errorf("can not find clusterprovider for cluster %q", seed.Name)
	}

	addonProvider := kubernetes.NewAddonProvider(
		fakeKubermaticImpersonationClient,
		sets.NewString("addon1", "addon2"),
	)
	addonProviders := map[string]provider.AddonProvider{"us-central1": addonProvider}
	addonProviderGetter := func(seed *kubermaticv1.Seed) (provider.AddonProvider, error) {
		if addonProvider, exists := addonProviders[seed.Name]; exists {
			return addonProvider, nil
		}
		return nil, fmt.Errorf("can not find addonprovider for cluster %q", seed.Name)
	}

	kubernetesInformerFactory.Start(wait.NeverStop)
	kubernetesInformerFactory.WaitForCacheSync(wait.NeverStop)
	kubermaticInformerFactory.Start(wait.NeverStop)
	kubermaticInformerFactory.WaitForCacheSync(wait.NeverStop)

	eventRecorderProvider := kubernetes.NewEventRecorder()

	// Disable the metrics endpoint in tests
	var prometheusClient prometheusapi.Client

	mainRouter := routingFunc(
		settingsProvider,
		userInfoGetter,
		seedsGetter,
		clusterProviderGetter,
		addonProviderGetter,
		sshKeyProvider,
		userProvider,
		serviceAccountProvider,
		serviceAccountTokenProvider,
		projectProvider,
		privilegedProjectProvider,
		fakeOIDCClient,
		tokenVerifiers,
		tokenExtractors,
		prometheusClient,
		projectMemberProvider,
		versions,
		updates,
		tokenAuth,
		tokenGenerator,
		eventRecorderProvider,
		credentialsManager,
	)

	return mainRouter, &ClientsSets{kubermaticClient, fakeClient, kubernetesClient, tokenAuth, tokenGenerator}, nil
}

// CreateTestEndpointAndGetClients is a convenience function that instantiates fake providers and sets up routes  for the tests
func CreateTestEndpointAndGetClients(user apiv1.User, seedsGetter provider.SeedsGetter, kubeObjects, machineObjects, kubermaticObjects []runtime.Object, versions []*version.Version, updates []*version.Update, routingFunc newRoutingFunc) (http.Handler, *ClientsSets, error) {
	credentialManager := presets.NewWithPresets(&kubermaticv1.PresetList{
		Items: []kubermaticv1.Preset{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "fake",
				},
				Spec: kubermaticv1.PresetSpec{
					RequiredEmailDomain: RequiredEmailDomain,
					Fake: &kubermaticv1.Fake{
						Token: "dummy_pluton_token",
					},
					Openstack: &kubermaticv1.Openstack{
						Username: TestOSuserName, Password: TestOSuserPass, Domain: TestOSdomain,
					},
				},
			},
		},
	})
	return initTestEndpoint(user, seedsGetter, kubeObjects, machineObjects, kubermaticObjects, versions, updates, credentialManager, routingFunc)
}

func CreateCredentialTestEndpoint(credentialsManager common.PresetsManager, routingFunc newRoutingFunc) (http.Handler, error) {
	apiUser := GenDefaultAPIUser()
	router, _, err := initTestEndpoint(*apiUser, nil, nil, nil, nil, nil, nil, credentialsManager, routingFunc)
	return router, err
}

// CreateTestEndpoint does exactly the same as CreateTestEndpointAndGetClients except it omits ClientsSets when returning
func CreateTestEndpoint(user apiv1.User, kubeObjects, kubermaticObjects []runtime.Object, versions []*version.Version, updates []*version.Update, routingFunc newRoutingFunc) (http.Handler, error) {
	router, _, err := CreateTestEndpointAndGetClients(user, nil, kubeObjects, nil, kubermaticObjects, versions, updates, routingFunc)
	return router, err
}

func buildSeeds() provider.SeedsGetter {
	return func() (map[string]*kubermaticv1.Seed, error) {
		return map[string]*kubermaticv1.Seed{
			"us-central1": {
				ObjectMeta: metav1.ObjectMeta{
					Name: "us-central1",
				},
				Spec: kubermaticv1.SeedSpec{
					Location: "us-central",
					Country:  "US",
					Datacenters: map[string]kubermaticv1.Datacenter{
						"private-do1": {
							Country:  "NL",
							Location: "US ",
							Spec: kubermaticv1.DatacenterSpec{
								Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
									Region: "ams2",
								},
							},
						},
						"regular-do1": {
							Country:  "NL",
							Location: "Amsterdam",
							Spec: kubermaticv1.DatacenterSpec{
								Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{
									Region: "ams2",
								},
							},
						},
						"restricted-fake-dc": {
							Country:  "NL",
							Location: "Amsterdam",
							Spec: kubermaticv1.DatacenterSpec{
								Fake:                &kubermaticv1.DatacenterSpecFake{},
								RequiredEmailDomain: "example.com",
							},
						},
						"restricted-fake-dc2": {
							Country:  "NL",
							Location: "Amsterdam",
							Spec: kubermaticv1.DatacenterSpec{
								Fake:                 &kubermaticv1.DatacenterSpecFake{},
								RequiredEmailDomains: []string{"23f67weuc.com", "example.com", "12noifsdsd.org"},
							},
						},
						"fake-dc": {
							Location: "Henriks basement",
							Country:  "Germany",
							Spec: kubermaticv1.DatacenterSpec{
								Fake: &kubermaticv1.DatacenterSpecFake{},
							},
						},
					},
				},
			},
		}, nil
	}
}

type fakeUserClusterConnection struct {
	fakeDynamicClient ctrlruntimeclient.Client
}

func (f *fakeUserClusterConnection) GetClient(_ *kubermaticv1.Cluster, _ ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return f.fakeDynamicClient, nil
}

func (f *fakeUserClusterConnection) GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error) {
	return []byte(generateTestKubeconfig(ClusterID, IDToken)), nil
}

func (f *fakeUserClusterConnection) GetViewerKubeconfig(c *kubermaticv1.Cluster) ([]byte, error) {
	return []byte(generateTestKubeconfig(ClusterID, IDViewerToken)), nil
}
func (f *fakeUserClusterConnection) RevokeViewerKubeconfig(c *kubermaticv1.Cluster) error {
	return nil
}

// ClientsSets a simple wrapper that holds fake client sets
type ClientsSets struct {
	FakeKubermaticClient *kubermaticfakeclentset.Clientset
	FakeClient           ctrlruntimeclient.Client
	// this client is used for unprivileged methods where impersonated client is used
	FakeKubernetesCoreClient kubernetesclientset.Interface

	TokenAuthenticator serviceaccount.TokenAuthenticator
	TokenGenerator     serviceaccount.TokenGenerator
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

// APIUserToKubermaticUser simply converts apiv1.User to kubermaticv1.User type
func APIUserToKubermaticUser(user apiv1.User) *kubermaticv1.User {
	var deletionTimestamp *metav1.Time
	if user.DeletionTimestamp != nil {
		deletionTimestamp = &metav1.Time{Time: user.DeletionTimestamp.Time}
	}
	return &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:              user.Name,
			CreationTimestamp: metav1.Time{Time: user.CreationTimestamp.Time},
			DeletionTimestamp: deletionTimestamp,
		},
		Spec: kubermaticv1.UserSpec{
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
func GenUser(id, name, email string) *kubermaticv1.User {
	if len(id) == 0 {
		// the name of the object is derived from the email address and encoded as sha256
		id = fmt.Sprintf("%x", sha256.Sum256([]byte(email)))
	}

	specID := ""
	{
		h := sha512.New512_224()
		if _, err := io.WriteString(h, email); err != nil {
			// not nice, better to use t.Error
			panic("unable to generate a test user due to " + err.Error())
		}
		specID = fmt.Sprintf("%x_KUBE", h.Sum(nil))
	}

	return &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
			UID:  types.UID(fmt.Sprintf("fake-uid-%s", id)),
		},
		Spec: kubermaticv1.UserSpec{
			ID:    specID,
			Name:  name,
			Email: email,
		},
	}
}

// GenInactiveServiceAccount generates a Service Account resource
func GenInactiveServiceAccount(id, name, group, projectName string) *kubermaticv1.User {
	user := GenUser(id, name, fmt.Sprintf("serviceaccount-%s@sa.kubermatic.io", id))
	user.Labels = map[string]string{kubernetes.ServiceAccountLabelGroup: fmt.Sprintf("%s-%s", group, projectName)}
	user.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.ProjectKindName,
			Name:       projectName,
			UID:        types.UID(id),
		},
	}
	user.Spec.ID = id
	user.Name = fmt.Sprintf("serviceaccount-%s", id)
	user.UID = ""

	return user
}

func GenServiceAccount(id, name, group, projectName string) *kubermaticv1.User {
	sa := GenInactiveServiceAccount(id, name, group, projectName)
	sa.Labels = map[string]string{}
	return sa
}

func GenDefaultServiceAccount() *kubermaticv1.User {
	return GenServiceAccount("1984", "default", "editors", GenDefaultProject().Name)
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
func GenDefaultUser() *kubermaticv1.User {
	userEmail := "bob@acme.com"
	return GenUser("", "Bob", userEmail)
}

// GenProject generates new empty project
func GenProject(name, phase string, creationTime time.Time, oRef ...metav1.OwnerReference) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:              fmt.Sprintf("%s-%s", name, "ID"),
			CreationTimestamp: metav1.NewTime(creationTime),
			OwnerReferences:   oRef,
		},
		Spec: kubermaticv1.ProjectSpec{Name: name},
		Status: kubermaticv1.ProjectStatus{
			Phase: phase,
		},
	}
}

// GenDefaultProject generates a default project
func GenDefaultProject() *kubermaticv1.Project {
	user := GenDefaultUser()
	oRef := metav1.OwnerReference{
		APIVersion: "kubermatic.io/v1",
		Kind:       "User",
		UID:        user.UID,
		Name:       user.Name,
	}
	return GenProject("my-first-project", kubermaticv1.ProjectActive, DefaultCreationTimestamp(), oRef)
}

// GenBinding generates a binding
func GenBinding(projectID, email, group string) *kubermaticv1.UserProjectBinding {
	return &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s-%s", projectID, email, group),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ProjectKindName,
					Name:       projectID,
				},
			},
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			UserEmail: email,
			ProjectID: projectID,
			Group:     fmt.Sprintf("%s-%s", group, projectID),
		},
	}
}

// GenDefaultOwnerBinding generates default owner binding
func GenDefaultOwnerBinding() *kubermaticv1.UserProjectBinding {
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

func GenCluster(id string, name string, projectID string, creationTime time.Time, modifiers ...func(*kubermaticv1.Cluster)) *kubermaticv1.Cluster {
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   id,
			Labels: map[string]string{"project-id": projectID},
			CreationTimestamp: func() metav1.Time {
				return metav1.NewTime(creationTime)
			}(),
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "FakeDatacenter",
				Fake:           &kubermaticv1.FakeCloudSpec{Token: "SecretToken"},
			},
			Version:           *semver.NewSemverOrDie("9.9.9"),
			HumanReadableName: name,
		},
		Address: kubermaticv1.ClusterAddress{
			AdminToken:   "drphc2.g4kq82pnlfqjqt65",
			ExternalName: "w225mx4z66.asia-east1-a-1.cloud.kubermatic.io",
			IP:           "35.194.142.199",
			URL:          "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
		},
		Status: kubermaticv1.ClusterStatus{
			ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
				Apiserver:                    kubermaticv1.HealthStatusUp,
				Scheduler:                    kubermaticv1.HealthStatusUp,
				Controller:                   kubermaticv1.HealthStatusUp,
				MachineController:            kubermaticv1.HealthStatusUp,
				Etcd:                         kubermaticv1.HealthStatusUp,
				UserClusterControllerManager: kubermaticv1.HealthStatusUp,
				CloudProviderInfrastructure:  kubermaticv1.HealthStatusUp,
			},
		},
	}

	for _, modifier := range modifiers {
		modifier(cluster)
	}
	return cluster
}

func GenDefaultCluster() *kubermaticv1.Cluster {
	return GenCluster(DefaultClusterID, DefaultClusterName, GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
}

func GenTestMachine(name, rawProviderSpec string, labels map[string]string, ownerRef []metav1.OwnerReference) *clusterv1alpha1.Machine {
	return &clusterv1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			UID:             types.UID(name + "-machine"),
			Name:            name,
			Namespace:       metav1.NamespaceSystem,
			Labels:          labels,
			OwnerReferences: ownerRef,
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "Machine",
		},
		Spec: clusterv1alpha1.MachineSpec{
			ProviderSpec: clusterv1alpha1.ProviderSpec{
				Value: &runtime.RawExtension{
					Raw: []byte(rawProviderSpec),
				},
			},
			Versions: clusterv1alpha1.MachineVersionInfo{
				Kubelet: "v9.9.9",
			},
		},
	}
}

func GenTestMachineDeployment(name, rawProviderSpec string, selector map[string]string) *clusterv1alpha1.MachineDeployment {
	var replicas int32 = 1
	return &clusterv1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceSystem,
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "MachineDeployment",
		},
		Spec: clusterv1alpha1.MachineDeploymentSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: selector,
			},
			Replicas: &replicas,
			Template: clusterv1alpha1.MachineTemplateSpec{
				Spec: clusterv1alpha1.MachineSpec{
					ProviderSpec: clusterv1alpha1.ProviderSpec{
						Value: &runtime.RawExtension{
							Raw: []byte(rawProviderSpec),
						},
					},
					Versions: clusterv1alpha1.MachineVersionInfo{
						Kubelet: "v9.9.9",
					},
				},
			},
		},
	}
}

func GenTestAddon(name string, variables *runtime.RawExtension, cluster *kubermaticv1.Cluster, creationTime time.Time) *kubermaticv1.Addon {
	if variables == nil {
		variables = &runtime.RawExtension{}
	}
	return &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ClusterKindName,
					Name:       cluster.Name,
					UID:        cluster.UID,
				},
			},
			CreationTimestamp: metav1.NewTime(creationTime),
		},
		Spec: kubermaticv1.AddonSpec{
			Name:      name,
			Variables: *variables,
			Cluster: corev1.ObjectReference{
				APIVersion: kubermaticv1.SchemeGroupVersion.String(),
				Kind:       kubermaticv1.ClusterKindName,
				Name:       cluster.Name,
				UID:        cluster.UID,
			},
		},
	}
}

func CheckStatusCode(wantStatusCode int, recorder *httptest.ResponseRecorder, t *testing.T) {
	t.Helper()
	if recorder.Code != wantStatusCode {
		t.Errorf("Expected status code to be %d, got: %d", wantStatusCode, recorder.Code)
		t.Error(recorder.Body.String())
		return
	}
}

func GenDefaultSaToken(projectID, saID, name, id string) *corev1.Secret {
	secret := &corev1.Secret{}
	secret.Name = fmt.Sprintf("sa-token-%s", id)
	secret.Type = "Opaque"
	secret.Namespace = "kubermatic"
	secret.Data = map[string][]byte{}
	secret.Data["token"] = []byte(TestFakeToken)
	secret.Labels = map[string]string{
		kubermaticv1.ProjectIDLabelKey: projectID,
		"name":                         name,
	}
	secret.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.UserKindName,
			UID:        "",
			Name:       saID,
		},
	}

	return secret
}

func GenDefaultExpiry() (apiv1.Time, error) {
	authenticator := serviceaccount.JWTTokenAuthenticator([]byte(TestServiceAccountHashKey))
	claim, _, err := authenticator.Authenticate(TestFakeToken)
	if err != nil {
		return apiv1.Time{}, err
	}
	return apiv1.NewTime(claim.Expiry.Time()), nil
}

func GenTestEvent(eventName, eventType, eventReason, eventMessage, kind, uid string) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      eventName,
			Namespace: metav1.NamespaceSystem,
		},
		InvolvedObject: corev1.ObjectReference{
			UID:       types.UID(uid),
			Name:      "testMachine",
			Namespace: metav1.NamespaceSystem,
			Kind:      kind,
		},
		Reason:  eventReason,
		Message: eventMessage,
		Source:  corev1.EventSource{Component: "eventTest"},
		Count:   1,
		Type:    eventType,
	}
}

func sortVersion(versions []*apiv1.MasterVersion) {
	sort.SliceStable(versions, func(i, j int) bool {
		mi, mj := versions[i], versions[j]
		return mi.Version.LessThan(mj.Version)
	})
}

func CompareVersions(t *testing.T, versions, expected []*apiv1.MasterVersion) {
	if len(versions) != len(expected) {
		t.Fatalf("got different lengths, got %d expected %d", len(versions), len(expected))
	}

	sortVersion(versions)
	sortVersion(expected)

	for i, version := range versions {
		if !version.Version.Equal(expected[i].Version) {
			t.Fatalf("expected version %v got %v", expected[i].Version, version.Version)
		}
		if version.Default != expected[i].Default {
			t.Fatalf("expected flag %v got %v", expected[i].Default, version.Default)
		}
	}
}
