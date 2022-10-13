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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	cluster.Spec = *genClusterSpec(name)
	cluster.Status = kubermaticv1.ClusterStatus{
		UserEmail:     userEmail,
		NamespaceName: kubernetes.NamespaceName(name),
	}

	return cluster
}
