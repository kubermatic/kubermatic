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
	"fmt"
	"time"

	constrainttemplatev1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// TestFakeFinalizer is a dummy finalizer with no special meaning.
	TestFakeFinalizer = "test.kubermatic.k8c.io/dummy"
)

func createAuthenitactedUser() *kubermaticv1.User {
	testUserName := "user1"
	testUserEmail := "john@acme.com"
	return &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: kubermaticv1.UserSpec{
			Name:  testUserName,
			Email: testUserEmail,
		},
	}
}

// genProject generates new empty project.
func genProject(name string, phase kubermaticv1.ProjectPhase, creationTime time.Time) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Project",
			APIVersion: "kubermatic.k8c.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              fmt.Sprintf("%s-%s", name, "ID"),
			CreationTimestamp: metav1.NewTime(creationTime),
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: name,
		},
		Status: kubermaticv1.ProjectStatus{
			Phase: phase,
		},
	}
}

// genDefaultProject generates a default project.
func genDefaultProject() *kubermaticv1.Project {
	return genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp())
}

// defaultCreationTimestamp returns default test timestamp.
func defaultCreationTimestamp() time.Time {
	return time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)
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
	cluster.Finalizers = []string{TestFakeFinalizer}
	cluster.Spec = *genClusterSpec(name)
	cluster.Status = kubermaticv1.ClusterStatus{
		UserEmail:     userEmail,
		NamespaceName: kubernetes.NamespaceName(name),
	}

	return cluster
}

func genConstraintTemplate(name string) *kubermaticv1.ConstraintTemplate {
	ct := &kubermaticv1.ConstraintTemplate{}
	ct.Kind = "ConstraintTemplate"
	ct.APIVersion = kubermaticv1.SchemeGroupVersion.String()
	ct.Name = name
	ct.Spec = kubermaticv1.ConstraintTemplateSpec{
		CRD: constrainttemplatev1.CRD{
			Spec: constrainttemplatev1.CRDSpec{
				Names: constrainttemplatev1.Names{
					Kind:       "labelconstraint",
					ShortNames: []string{"lc"},
				},
			},
		},
		Targets: []constrainttemplatev1.Target{
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
