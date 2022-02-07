/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubernetes_test

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	"gopkg.in/square/go-jose.v2/jwt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/cloud"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/serviceaccount"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// TestFakeToken signed JWT token with fake data.
	TestFakeToken = "eyJhbGciOiJIUzI1NiJ9.eyJlbWFpbCI6IjEiLCJleHAiOjE2NDk3NDg4NTYsImlhdCI6MTU1NTA1NDQ1NiwibmJmIjoxNTU1MDU0NDU2LCJwcm9qZWN0X2lkIjoiMSIsInRva2VuX2lkIjoiMSJ9.Q4qxzOaCvUnWfXneY654YiQjUTd_Lsmw56rE17W2ouo"

	// TestFakeFinalizer is a dummy finalizer with no special meaning.
	TestFakeFinalizer = "test.kubermatic.k8c.io/dummy"
)

type fakeJWTTokenGenerator struct {
}

// Generate generates new fake token.
func (j *fakeJWTTokenGenerator) Generate(claims *jwt.Claims, privateClaims *serviceaccount.CustomTokenClaim) (string, error) {
	return TestFakeToken, nil
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

func createProjectSA(name, projectName, group, id string) *kubermaticv1.User {
	sa := genProjectServiceAccount(id, name, group, projectName)
	// remove autogenerated values
	sa.OwnerReferences[0].UID = ""
	sa.Spec.Email = ""
	sa.Spec.ID = ""

	return sa
}

func createSANoPrefix(name, projectName, group, id string) *kubermaticv1.User {
	sa := createProjectSA(name, projectName, group, id)
	sa.Name = strings.TrimPrefix(sa.Name, "serviceaccount-")
	return sa
}

func sortTokenByName(tokens []*corev1.Secret) {
	sort.SliceStable(tokens, func(i, j int) bool {
		mi, mj := tokens[i], tokens[j]
		return mi.Name < mj.Name
	})
}

// genUser generates a User resource
// note if the id is empty then it will be auto generated.
func genUser(id, name, email string) *kubermaticv1.User {
	if len(id) == 0 {
		// the name of the object is derived from the email address and encoded as sha256
		id = fmt.Sprintf("%x", sha256.Sum256([]byte(email)))
	}

	h := sha512.New512_224()
	if _, err := io.WriteString(h, email); err != nil {
		// not nice, better to use t.Error
		panic("unable to generate a test user: " + err.Error())
	}
	specID := fmt.Sprintf("%x_KUBE", h.Sum(nil))

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

// genDefaultUser generates a default user.
func genDefaultUser() *kubermaticv1.User {
	userEmail := "bob@acme.com"
	return genUser("", "Bob", userEmail)
}

// genProject generates new empty project.
func genProject(name string, phase kubermaticv1.ProjectPhase, creationTime time.Time, oRef ...metav1.OwnerReference) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Project",
			APIVersion: "kubermatic.k8c.io/v1",
		},
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

// genDefaultProject generates a default project.
func genDefaultProject() *kubermaticv1.Project {
	user := genDefaultUser()
	oRef := metav1.OwnerReference{
		APIVersion: "kubermatic.k8c.io/v1",
		Kind:       "User",
		UID:        user.UID,
		Name:       user.Name,
	}
	return genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp(), oRef)
}

// defaultCreationTimestamp returns default test timestamp.
func defaultCreationTimestamp() time.Time {
	return time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)
}

// genProjectServiceAccount generates a Service Account resource.
func genProjectServiceAccount(id, name, group, projectName string) *kubermaticv1.User {
	user := genUser(id, name, fmt.Sprintf("serviceaccount-%s@sa.kubermatic.io", id))
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

// genBinding generates a binding.
func genBinding(projectID, email, group string) *kubermaticv1.UserProjectBinding {
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

func genSecret(projectID, saID, name, id string) *corev1.Secret {
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

func genClusterSpec(name string) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: "FakeDatacenter",
			Fake:           &kubermaticv1.FakeCloudSpec{Token: "SecretToken"},
		},
		HumanReadableName: name,
	}
}

func genCluster(name, clusterType, projectID, workerName, userEmail string) *kubermaticv1.Cluster {
	cluster := &kubermaticv1.Cluster{}

	labels := map[string]string{
		kubermaticv1.ProjectIDLabelKey: projectID,
	}
	if len(workerName) > 0 {
		labels[kubermaticv1.WorkerNameLabelKey] = workerName
	}

	cluster.Labels = labels
	cluster.Name = name
	cluster.Status = kubermaticv1.ClusterStatus{
		UserEmail:              userEmail,
		NamespaceName:          fmt.Sprintf("cluster-%s", name),
		CloudMigrationRevision: cloud.CurrentMigrationRevision,
	}
	cluster.Address = kubermaticv1.ClusterAddress{}
	cluster.Finalizers = []string{TestFakeFinalizer}
	cluster.Status.ExtendedHealth = kubermaticv1.ExtendedClusterHealth{
		Apiserver:                    kubermaticv1.HealthStatusProvisioning,
		Scheduler:                    kubermaticv1.HealthStatusProvisioning,
		Controller:                   kubermaticv1.HealthStatusProvisioning,
		MachineController:            kubermaticv1.HealthStatusProvisioning,
		Etcd:                         kubermaticv1.HealthStatusProvisioning,
		OpenVPN:                      kubermaticv1.HealthStatusProvisioning,
		CloudProviderInfrastructure:  kubermaticv1.HealthStatusProvisioning,
		UserClusterControllerManager: kubermaticv1.HealthStatusProvisioning,
	}

	cluster.Spec = *genClusterSpec(name)
	return cluster
}

func genConstraintTemplate(name string) *kubermaticv1.ConstraintTemplate {
	ct := &kubermaticv1.ConstraintTemplate{}
	ct.Kind = "ConstraintTemplate"
	ct.APIVersion = kubermaticv1.SchemeGroupVersion.String()
	ct.Name = name
	ct.Spec = kubermaticv1.ConstraintTemplateSpec{
		CRD: v1beta1.CRD{
			Spec: v1beta1.CRDSpec{
				Names: v1beta1.Names{
					Kind:       "labelconstraint",
					ShortNames: []string{"lc"},
				},
			},
		},
		Targets: []v1beta1.Target{
			{
				Target: "admission.k8s.gatekeeper.sh",
				Rego: `
		package k8srequiredlabels

        deny[{"msg": msg, "details": {"missing_labels": missing}}] {
          provided := {label | input.review.object.metadata.labels[label]}
          required := {label | label := input.parameters.labels[_]}
          missing := required - provided
          count(missing) > 0
          msg := sprintf("you must provide labels: %v", [missing])
        }`,
			},
		},
		Selector: kubermaticv1.ConstraintTemplateSelector{
			Providers: []string{"aws", "gcp"},
			LabelSelector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "cluster",
						Operator: metav1.LabelSelectorOpExists,
					},
				},
				MatchLabels: map[string]string{
					"deployment": "prod",
					"domain":     "sales",
				},
			},
		},
	}

	return ct
}
