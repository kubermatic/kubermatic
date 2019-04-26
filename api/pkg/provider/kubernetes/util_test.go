package kubernetes_test

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"k8s.io/apimachinery/pkg/types"

	kubermaticfakeclentset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"
	"gopkg.in/square/go-jose.v2/jwt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesclient "k8s.io/client-go/kubernetes"
	fakerestclient "k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	// TestFakeToken signed JWT token with fake data
	TestFakeToken = "eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6IjEiLCJleHAiOjE2NDk3NDg4NTYsImlhdCI6MTU1NTA1NDQ1NiwibmJmIjoxNTU1MDU0NDU2LCJwcm9qZWN0X2lkIjoiMSIsInRva2VuX2lkIjoiMSJ9.Q4qxzOaCvUnWfXneY654YiQjUTd_Lsmw56rE17W2ouo"
)

// fakeKubernetesImpersonationClient gives kubernetes client set that uses user impersonation
type fakeKubernetesImpersonationClient struct {
	kubernetesClent *fakerestclient.Clientset
}

func (f *fakeKubernetesImpersonationClient) CreateKubernetesFakeImpersonatedClientSet(impCfg restclient.ImpersonationConfig) (kubernetesclient.Interface, error) {
	return f.kubernetesClent, nil
}

func createFakeKubernetesClients(kubermaticObjects []runtime.Object) (*fakeKubernetesImpersonationClient, *fakerestclient.Clientset, cache.Indexer, error) {
	kubermaticClient := fakerestclient.NewSimpleClientset(kubermaticObjects...)

	indexer, err := createIndexer(kubermaticObjects)
	if err != nil {
		return nil, nil, nil, err
	}
	return &fakeKubernetesImpersonationClient{kubermaticClient}, kubermaticClient, indexer, nil
}

type fakeJWTTokenGenerator struct {
}

// Generate generates new fake token
func (j *fakeJWTTokenGenerator) Generate(claims *jwt.Claims, privateClaims *serviceaccount.CustomTokenClaim) (string, error) {
	return TestFakeToken, nil
}

// fakeKubermaticImpersonationClient gives kubermatic client set that uses user impersonation
type fakeKubermaticImpersonationClient struct {
	kubermaticClent *kubermaticfakeclentset.Clientset
}

func (f *fakeKubermaticImpersonationClient) CreateFakeImpersonatedClientSet(impCfg restclient.ImpersonationConfig) (kubermaticclientv1.KubermaticV1Interface, error) {
	return f.kubermaticClent.KubermaticV1(), nil
}

func createFakeKubermaticClients(kubermaticObjects []runtime.Object) (*fakeKubermaticImpersonationClient, *kubermaticfakeclentset.Clientset, cache.Indexer, error) {
	kubermaticClient := kubermaticfakeclentset.NewSimpleClientset(kubermaticObjects...)

	indexer, err := createIndexer(kubermaticObjects)
	if err != nil {
		return nil, nil, nil, err
	}
	return &fakeKubermaticImpersonationClient{kubermaticClient}, kubermaticClient, indexer, nil
}

func createIndexer(kubermaticObjects []runtime.Object) (cache.Indexer, error) {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	for _, obj := range kubermaticObjects {
		err := indexer.Add(obj)
		if err != nil {
			return nil, err
		}
	}
	return indexer, nil
}

func createAuthenitactedUser() *kubermaticv1.User {
	testUserID := "1233"
	testUserName := "user1"
	testUserEmail := "john@acme.com"
	return &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: kubermaticv1.UserSpec{
			Name:  testUserName,
			ID:    testUserID,
			Email: testUserEmail,
		},
	}
}

func createBinding(name, projectID, email, group string) *kubermaticv1.UserProjectBinding {
	binding := genBinding(projectID, email, group)
	binding.Kind = kubermaticv1.UserProjectBindingKind
	binding.Name = name
	return binding
}

func createSA(name, projectName, group, id string) *kubermaticv1.User {
	sa := genServiceAccount(id, name, group, projectName)
	// remove autogenerated values
	sa.OwnerReferences[0].UID = ""
	sa.Spec.Email = ""
	sa.Spec.ID = ""

	return sa
}

func createSANoPrefix(name, projectName, group, id string) *kubermaticv1.User {
	sa := createSA(name, projectName, group, id)
	sa.Name = strings.TrimPrefix(sa.Name, "serviceaccount-")
	return sa
}

func sortTokenByName(tokens []*v1.Secret) {
	sort.SliceStable(tokens, func(i, j int) bool {
		mi, mj := tokens[i], tokens[j]
		return mi.Name < mj.Name
	})
}

// genUser generates a User resource
// note if the id is empty then it will be auto generated
func genUser(id, name, email string) *kubermaticapiv1.User {
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

	return &kubermaticapiv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
			UID:  types.UID(fmt.Sprintf("fake-uid-%s", id)),
		},
		Spec: kubermaticapiv1.UserSpec{
			ID:    specID,
			Name:  name,
			Email: email,
		},
	}
}

// genDefaultUser generates a default user
func genDefaultUser() *kubermaticapiv1.User {
	userEmail := "bob@acme.com"
	return genUser("", "Bob", userEmail)
}

// genProject generates new empty project
func genProject(name, phase string, creationTime time.Time, oRef ...metav1.OwnerReference) *kubermaticapiv1.Project {
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

// genDefaultProject generates a default project
func genDefaultProject() *kubermaticapiv1.Project {
	user := genDefaultUser()
	oRef := metav1.OwnerReference{
		APIVersion: "kubermatic.io/v1",
		Kind:       "User",
		UID:        user.UID,
		Name:       user.Name,
	}
	return genProject("my-first-project", kubermaticapiv1.ProjectActive, defaultCreationTimestamp(), oRef)
}

// defaultCreationTimestamp returns default test timestamp
func defaultCreationTimestamp() time.Time {
	return time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)
}

// genServiceAccount generates a Service Account resource
func genServiceAccount(id, name, group, projectName string) *kubermaticapiv1.User {
	user := genUser(id, name, fmt.Sprintf("serviceaccount-%s@sa.kubermatic.io", id))
	user.Labels = map[string]string{kubernetes.ServiceAccountLabelGroup: fmt.Sprintf("%s-%s", group, projectName)}
	user.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
			Kind:       kubermaticapiv1.ProjectKindName,
			Name:       projectName,
			UID:        types.UID(id),
		},
	}
	user.Spec.ID = id
	user.Name = fmt.Sprintf("serviceaccount-%s", id)
	user.UID = ""

	return user
}

// genBinding generates a binding
func genBinding(projectID, email, group string) *kubermaticapiv1.UserProjectBinding {
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

func genSecret(projectID, saID, name, id string) *v1.Secret {
	secret := &v1.Secret{}
	secret.Name = fmt.Sprintf("sa-token-%s", id)
	secret.Type = "Opaque"
	secret.Namespace = "kubermatic"
	secret.Data = map[string][]byte{}
	secret.Data["token"] = []byte(TestFakeToken)
	secret.Labels = map[string]string{
		kubermaticapiv1.ProjectIDLabelKey: projectID,
		"name":                            name,
	}
	secret.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
			Kind:       kubermaticapiv1.UserKindName,
			UID:        "",
			Name:       saID,
		},
	}

	return secret
}

func genClusterSpec(name string) *kubermaticapiv1.ClusterSpec {
	return &kubermaticapiv1.ClusterSpec{
		Cloud: kubermaticapiv1.CloudSpec{
			DatacenterName: "FakeDatacenter",
			Fake:           &kubermaticapiv1.FakeCloudSpec{Token: "SecretToken"},
		},
		HumanReadableName: name,
	}
}

func genCluster(name, clusterType, projectID, workerName, userEmail string) *kubermaticapiv1.Cluster {
	cluster := &kubermaticapiv1.Cluster{}

	labels := map[string]string{
		kubermaticapiv1.ProjectIDLabelKey: projectID,
	}
	if len(workerName) > 0 {
		labels[kubermaticapiv1.WorkerNameLabelKey] = workerName
	}

	cluster.Labels = labels
	cluster.Name = name
	cluster.Status = kubermaticapiv1.ClusterStatus{
		UserEmail:     userEmail,
		NamespaceName: fmt.Sprintf("cluster-%s", name),
	}
	cluster.Address = kubermaticapiv1.ClusterAddress{}

	if clusterType == "openshift" {
		cluster.Annotations = map[string]string{
			"kubermatic.io/openshift": "true",
		}
	}
	cluster.Spec = *genClusterSpec(name)
	return cluster
}
