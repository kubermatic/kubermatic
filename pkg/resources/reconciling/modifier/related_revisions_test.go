/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package modifier

import (
	"context"
	"testing"

	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRelatedRevisions(t *testing.T) {
	existingLabels := map[string]string{
		"foo":   "bar",
		"hello": "world",
	}

	// keep a copy as a reference for later
	referenceLabels := map[string]string{}
	for k, v := range existingLabels {
		referenceLabels[k] = v
	}

	const (
		namespace  = "cluster-xyz"
		secretName = "test-data"
	)

	deployment := &appsv1.Deployment{}
	deployment.Name = "test"
	deployment.Namespace = namespace

	// This is the important bit: Both the object metadata and the selector use the
	// *identical* map. The modifier must however only change one of them.
	deployment.Spec.Template.Labels = existingLabels
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: existingLabels,
	}

	deployment.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name: "test",
			EnvFrom: []corev1.EnvFromSource{
				{
					SecretRef: &corev1.SecretEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretName,
						},
					},
				},
			},
		},
	}

	client := fake.
		NewClientBuilder().
		WithObjects(
			deployment,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "123456",
					Name:            secretName,
					Namespace:       namespace,
				},
			},
		).
		Build()

	ctx := context.Background()

	modified, err := addRevisionLabelsToObject(ctx, client, deployment)
	if err != nil {
		t.Fatalf("Failed to add revisions: %v", err)
	}

	modDeployment := modified.(*appsv1.Deployment)

	// ensure that a new label has been added to the PodTemplate
	if labels := len(modDeployment.Spec.Template.Labels); labels != len(referenceLabels)+1 {
		t.Errorf("Expected a new label (%d in total) in the PodTemplate's object meta, but got %d", len(referenceLabels)+1, labels)
	}

	// ensure that the MatchLabels have *not* changed
	if changes := diff.ObjectDiff(referenceLabels, modDeployment.Spec.Selector.MatchLabels); changes != "" {
		t.Errorf("MatchLabels has changed, but should have remained constant:\n\n%s", changes)
	}
}
