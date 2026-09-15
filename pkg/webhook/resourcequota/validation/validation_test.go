/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package validation

import (
	"context"
	"strings"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testProjectName = "project-id"
	testProjectUID  = types.UID("project-uid")
)

type accountingValidatorRoute struct {
	name      string
	validator admission.Validator[*kubermaticv1.ResourceQuota]
}

func accountingValidatorRoutes(client ctrlruntimeclient.Client, enabledFeatures features.FeatureGate) []accountingValidatorRoute {
	return []accountingValidatorRoute{
		{name: "general route", validator: NewValidator(client, enabledFeatures)},
		{name: "dedicated route", validator: NewAcceleratorAccountingValidator(client, enabledFeatures)},
	}
}

func TestAccountingAnnotationRejectedOnCreate(t *testing.T) {
	validator := NewValidator(fake.NewClientBuilder().Build(), features.FeatureGate{
		features.KubeVirtAcceleratorQuota: true,
	})

	for _, value := range []string{resources.AcceleratorAccountingEnabledAnnotationValue, "false", ""} {
		t.Run("value="+value, func(t *testing.T) {
			quota := testResourceQuota(testProject())
			quota.Annotations = map[string]string{resources.AcceleratorAccountingEnabledAnnotation: value}

			_, err := validator.ValidateCreate(context.Background(), quota)
			assertErrorContains(t, err, "cannot be set during creation")
		})
	}
}

func TestDedicatedAccountingValidatorRejectsAnnotatedCreate(t *testing.T) {
	validator := NewAcceleratorAccountingValidator(fake.NewClientBuilder().Build(), features.FeatureGate{
		features.KubeVirtAcceleratorQuota: true,
	})
	quota := testResourceQuota(testProject())
	quota.Annotations = map[string]string{resources.AcceleratorAccountingEnabledAnnotation: resources.AcceleratorAccountingEnabledAnnotationValue}

	_, err := validator.ValidateCreate(context.Background(), quota)
	assertErrorContains(t, err, "cannot be set during creation")
}

func TestAccountingActivationUpdate(t *testing.T) {
	if !acceleratorAccountingSupported() {
		project := testProject()
		oldQuota := testResourceQuota(project)
		newQuota := oldQuota.DeepCopy()
		newQuota.Annotations = map[string]string{
			resources.AcceleratorAccountingEnabledAnnotation: resources.AcceleratorAccountingEnabledAnnotationValue,
		}

		client := fake.NewClientBuilder().WithObjects(project).Build()
		enabledFeatures := features.FeatureGate{
			features.KubeVirtAcceleratorQuota: true,
		}
		for _, route := range accountingValidatorRoutes(client, enabledFeatures) {
			t.Run(route.name, func(t *testing.T) {
				_, err := route.validator.ValidateUpdate(context.Background(), oldQuota, newQuota)
				assertErrorContains(t, err, "only supported in Kubermatic Enterprise Edition")
			})
		}
		return
	}

	testCases := []struct {
		name           string
		featureEnabled bool
		missingProject bool
		mutateProject  func(*kubermaticv1.Project)
		mutateOld      func(*kubermaticv1.ResourceQuota)
		mutateNew      func(*kubermaticv1.ResourceQuota)
		errorContains  string
	}{
		{
			name:           "prepared project quota is activated",
			featureEnabled: true,
		},
		{
			name:           "annotation value must be exact",
			featureEnabled: true,
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				quota.Annotations[resources.AcceleratorAccountingEnabledAnnotation] = "True"
			},
			errorContains: "must have the exact value",
		},
		{
			name:          "feature gate is required",
			errorContains: "feature gate",
		},
		{
			name:           "only project quota is supported",
			featureEnabled: true,
			mutateOld: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Subject.Kind = "seed"
			},
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Subject.Kind = "seed"
			},
			errorContains: "only supported for project quotas",
		},
		{
			name:           "default-managed project quota must be detached before activation",
			featureEnabled: true,
			mutateOld: func(quota *kubermaticv1.ResourceQuota) {
				quota.Labels["kkp-default-resource-quota"] = "true"
			},
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				quota.Labels["kkp-default-resource-quota"] = "true"
			},
			errorContains: "default-managed ResourceQuota",
		},
		{
			name:           "default-management label removal and activation must be separate updates",
			featureEnabled: true,
			mutateOld: func(quota *kubermaticv1.ResourceQuota) {
				quota.Labels["kkp-default-resource-quota"] = "true"
			},
			errorContains: "default-managed ResourceQuota",
		},
		{
			name:           "accelerator limits must initially be empty",
			featureEnabled: true,
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators = testAcceleratorQuota()
			},
			errorContains: "accelerator limits must be empty",
		},
		{
			name:           "project must exist",
			featureEnabled: true,
			missingProject: true,
			errorContains:  "does not exist",
		},
		{
			name:           "project phase must not be terminating",
			featureEnabled: true,
			mutateProject: func(project *kubermaticv1.Project) {
				project.Status.Phase = kubermaticv1.ProjectTerminating
			},
			errorContains: "project \"project-id\" is terminating",
		},
		{
			name:           "deleting project is terminating even before phase update",
			featureEnabled: true,
			mutateProject: func(project *kubermaticv1.Project) {
				now := metav1.Now()
				project.DeletionTimestamp = &now
				project.Finalizers = []string{"test.kubermatic.io/project-cleanup"}
			},
			errorContains: "project \"project-id\" is terminating",
		},
		{
			name:           "subject labels must already match",
			featureEnabled: true,
			mutateOld: func(quota *kubermaticv1.ResourceQuota) {
				delete(quota.Labels, kubermaticv1.ResourceQuotaSubjectNameLabelKey)
			},
			errorContains: "must already have subject labels",
		},
		{
			name:           "subject labels must be retained",
			featureEnabled: true,
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				quota.Labels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] = "other"
			},
			errorContains: "must retain subject labels",
		},
		{
			name:           "project controller owner must already exist",
			featureEnabled: true,
			mutateOld: func(quota *kubermaticv1.ResourceQuota) {
				quota.OwnerReferences = nil
			},
			errorContains: "must already have a controller owner reference",
		},
		{
			name:           "project controller owner must be retained",
			featureEnabled: true,
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				quota.OwnerReferences = nil
			},
			errorContains: "must retain its controller owner reference",
		},
		{
			name:           "owner UID must match live project",
			featureEnabled: true,
			mutateOld: func(quota *kubermaticv1.ResourceQuota) {
				quota.OwnerReferences[0].UID = "different-project-uid"
			},
			errorContains: "must already have a controller owner reference",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			project := testProject()
			if tc.mutateProject != nil {
				tc.mutateProject(project)
			}

			oldQuota := testResourceQuota(project)
			newQuota := oldQuota.DeepCopy()
			newQuota.Annotations = map[string]string{
				resources.AcceleratorAccountingEnabledAnnotation: resources.AcceleratorAccountingEnabledAnnotationValue,
			}
			if tc.mutateOld != nil {
				tc.mutateOld(oldQuota)
			}
			if tc.mutateNew != nil {
				tc.mutateNew(newQuota)
			}

			objects := []ctrlruntimeclient.Object{}
			if !tc.missingProject {
				objects = append(objects, project)
			}
			client := fake.NewClientBuilder().WithObjects(objects...).Build()
			enabledFeatures := features.FeatureGate{
				features.KubeVirtAcceleratorQuota: tc.featureEnabled,
			}

			for _, route := range accountingValidatorRoutes(client, enabledFeatures) {
				t.Run(route.name, func(t *testing.T) {
					_, err := route.validator.ValidateUpdate(context.Background(), oldQuota, newQuota)
					if tc.errorContains == "" {
						if err != nil {
							t.Fatalf("expected activation to succeed, got %v", err)
						}
						return
					}
					assertErrorContains(t, err, tc.errorContains)
				})
			}
		})
	}
}

func TestEnabledAccountingAnnotationIsImmutable(t *testing.T) {
	oldQuota := testResourceQuota(testProject())
	oldQuota.Annotations = map[string]string{
		resources.AcceleratorAccountingEnabledAnnotation: resources.AcceleratorAccountingEnabledAnnotationValue,
	}

	type testCase struct {
		name          string
		mutate        func(*kubermaticv1.ResourceQuota)
		errorContains string
	}
	testCases := []testCase{
		{
			name:   "unchanged annotation remains valid without gate or project lookup",
			mutate: func(*kubermaticv1.ResourceQuota) {},
		},
		{
			name: "project controller owner cannot be removed",
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				quota.OwnerReferences = nil
			},
			errorContains: "must retain its project controller owner reference",
		},
		{
			name: "project subject labels cannot be removed",
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				delete(quota.Labels, kubermaticv1.ResourceQuotaSubjectNameLabelKey)
			},
			errorContains: "must retain subject labels",
		},
		{
			name: "project subject cannot be changed with matching labels",
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Subject.Name = "another-project"
				quota.Labels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] = "another-project"
			},
			errorContains: "project subject is immutable",
		},
		{
			name: "accelerator limits require current readiness",
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators = testAcceleratorQuota()
			},
			errorContains: "until accelerator accounting is ready",
		},
		{
			name: "annotation cannot be removed",
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				delete(quota.Annotations, resources.AcceleratorAccountingEnabledAnnotation)
			},
			errorContains: "is immutable once accelerator accounting is enabled",
		},
		{
			name: "annotation cannot be changed",
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				quota.Annotations[resources.AcceleratorAccountingEnabledAnnotation] = "false"
			},
			errorContains: "is immutable once accelerator accounting is enabled",
		},
	}
	if acceleratorAccountingSupported() {
		testCases = append(testCases, testCase{
			name: "default-management label cannot be added after activation",
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				quota.Labels["kkp-default-resource-quota"] = "true"
			},
			errorContains: "default-managed ResourceQuota",
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, route := range accountingValidatorRoutes(fake.NewClientBuilder().Build(), nil) {
				t.Run(route.name, func(t *testing.T) {
					newQuota := oldQuota.DeepCopy()
					tc.mutate(newQuota)

					_, err := route.validator.ValidateUpdate(context.Background(), oldQuota, newQuota)
					if tc.errorContains == "" {
						if err != nil {
							t.Fatalf("expected update to succeed, got %v", err)
						}
						return
					}
					assertErrorContains(t, err, tc.errorContains)
				})
			}
		})
	}
}

func TestEnabledAccountingAcceleratorQuotaTransitions(t *testing.T) {
	oldQuota := testResourceQuota(testProject())
	oldQuota.Annotations = map[string]string{
		resources.AcceleratorAccountingEnabledAnnotation: resources.AcceleratorAccountingEnabledAnnotationValue,
	}

	tests := []struct {
		name          string
		prepareOld    func(*kubermaticv1.ResourceQuota)
		mutateNew     func(*kubermaticv1.ResourceQuota)
		errorContains string
	}{
		{
			name: "first non-empty limits are accepted after empty accounting is ready",
			prepareOld: func(quota *kubermaticv1.ResourceQuota) {
				setReadyGlobalAcceleratorAccounting(quota, metav1.Now())
			},
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators = testAcceleratorQuota()
			},
		},
		{
			name: "accelerator change is accepted after current limits are ready",
			prepareOld: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators = testAcceleratorQuota()
				setReadyGlobalAcceleratorAccounting(quota, metav1.Now())
			},
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators[0].Resources["nvidia.com/GH100_H200_NVL"] = resource.MustParse("3")
			},
		},
		{
			name: "unchanged accelerator limits allow scalar edits while blocked",
			prepareOld: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators = testAcceleratorQuota()
			},
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				cpu := resource.MustParse("12")
				quota.Spec.Quota.CPU = &cpu
			},
		},
		{
			name: "remove all is always available as recovery",
			prepareOld: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators = testAcceleratorQuota()
			},
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators = nil
			},
		},
		{
			name: "blocked accounting rejects non-empty transition",
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators = testAcceleratorQuota()
			},
			errorContains: "until accelerator accounting is ready",
		},
		{
			name: "stale accounting rejects non-empty transition",
			prepareOld: func(quota *kubermaticv1.ResourceQuota) {
				setReadyGlobalAcceleratorAccounting(quota, metav1.NewTime(time.Now().Add(-resources.AcceleratorAccountingHeartbeatTimeout-time.Second)))
			},
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators = testAcceleratorQuota()
			},
			errorContains: "readiness is stale",
		},
		{
			name: "digest mismatch serializes transitions",
			prepareOld: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators = testAcceleratorQuota()
				setReadyGlobalAcceleratorAccounting(quota, metav1.Now())
				quota.Status.GlobalAcceleratorAccounting.ObservedQuotaDigest = "sha256:previous"
			},
			mutateNew: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators[0].Resources["nvidia.com/GH100_H200_NVL"] = resource.MustParse("3")
			},
			errorContains: "does not match the current quota revision and digest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			old := oldQuota.DeepCopy()
			if tc.prepareOld != nil {
				tc.prepareOld(old)
			}
			updated := old.DeepCopy()
			if tc.mutateNew != nil {
				tc.mutateNew(updated)
			}

			for _, route := range accountingValidatorRoutes(fake.NewClientBuilder().Build(), nil) {
				t.Run(route.name, func(t *testing.T) {
					_, err := route.validator.ValidateUpdate(context.Background(), old, updated)
					if tc.errorContains == "" {
						if err != nil {
							t.Fatalf("expected update to succeed, got %v", err)
						}
						return
					}
					assertErrorContains(t, err, tc.errorContains)
				})
			}
		})
	}
}

func setReadyGlobalAcceleratorAccounting(quota *kubermaticv1.ResourceQuota, observedAt metav1.Time) {
	quota.Status.GlobalAcceleratorAccounting = &kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus{
		ActivationPhase:            kubermaticv1.AcceleratorAccountingPhaseReady,
		ObservedAccountingRevision: "revision-1",
		ObservedQuotaDigest:        kubermaticv1.AcceleratorQuotaDigestFor(quota.Spec.Quota.Accelerators),
		ObservedAt:                 observedAt,
		Ready:                      true,
	}
}

func TestMalformedAccountingAnnotationCanOnlyBeRemoved(t *testing.T) {
	for _, route := range accountingValidatorRoutes(fake.NewClientBuilder().Build(), nil) {
		t.Run(route.name, func(t *testing.T) {
			oldQuota := testResourceQuota(testProject())
			oldQuota.Annotations = map[string]string{resources.AcceleratorAccountingEnabledAnnotation: "false"}

			unchanged := oldQuota.DeepCopy()
			_, err := route.validator.ValidateUpdate(context.Background(), oldQuota, unchanged)
			assertErrorContains(t, err, "has invalid value")

			removed := oldQuota.DeepCopy()
			delete(removed.Annotations, resources.AcceleratorAccountingEnabledAnnotation)
			if _, err := route.validator.ValidateUpdate(context.Background(), oldQuota, removed); err != nil {
				t.Fatalf("expected invalid annotation removal to succeed, got %v", err)
			}

			deleting := oldQuota.DeepCopy()
			now := metav1.Now()
			deleting.DeletionTimestamp = &now
			if _, err := route.validator.ValidateUpdate(context.Background(), oldQuota, deleting); err != nil {
				t.Fatalf("expected deleting malformed quota cleanup to succeed, got %v", err)
			}
		})
	}
}

func TestActiveAccountingResourceQuotaDelete(t *testing.T) {
	activeProject := testProject()
	terminatingProject := testProject()
	now := metav1.Now()
	terminatingProject.DeletionTimestamp = &now
	terminatingProject.Finalizers = []string{"test.kubermatic.io/project-cleanup"}

	testCases := []struct {
		name          string
		project       *kubermaticv1.Project
		errorContains string
	}{
		{
			name:          "active project keeps activation anchor",
			project:       activeProject,
			errorContains: "cannot be deleted while project",
		},
		{
			name:    "terminating project may delete activation anchor",
			project: terminatingProject,
		},
		{
			name: "missing project may finish garbage collection",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			quota := testResourceQuota(testProject())
			quota.Annotations = map[string]string{
				resources.AcceleratorAccountingEnabledAnnotation: resources.AcceleratorAccountingEnabledAnnotationValue,
			}
			objects := []ctrlruntimeclient.Object{}
			if tc.project != nil {
				objects = append(objects, tc.project)
			}
			client := fake.NewClientBuilder().WithObjects(objects...).Build()
			for _, route := range accountingValidatorRoutes(client, nil) {
				t.Run(route.name, func(t *testing.T) {
					_, err := route.validator.ValidateDelete(context.Background(), quota)
					if tc.errorContains == "" {
						if err != nil {
							t.Fatalf("expected delete to succeed, got %v", err)
						}
						return
					}
					assertErrorContains(t, err, tc.errorContains)
				})
			}
		})
	}
}

func testProject() *kubermaticv1.Project {
	return &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: testProjectName,
			UID:  testProjectUID,
		},
		Status: kubermaticv1.ProjectStatus{Phase: kubermaticv1.ProjectActive},
	}
}

func testResourceQuota(project *kubermaticv1.Project) *kubermaticv1.ResourceQuota {
	return &kubermaticv1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: "project-quota",
			Labels: map[string]string{
				kubermaticv1.ResourceQuotaSubjectNameLabelKey: project.Name,
				kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(project, schema.GroupVersionKind{
					Group:   kubermaticv1.SchemeGroupVersion.Group,
					Version: kubermaticv1.SchemeGroupVersion.Version,
					Kind:    kubermaticv1.ProjectKindName,
				}),
			},
		},
		Spec: kubermaticv1.ResourceQuotaSpec{
			Subject: kubermaticv1.Subject{
				Name: project.Name,
				Kind: kubermaticv1.ProjectSubjectKind,
			},
			Quota: kubermaticv1.ResourceDetails{},
		},
	}
}

func testAcceleratorQuota() []kubermaticv1.AcceleratorQuota {
	return []kubermaticv1.AcceleratorQuota{{
		Provider: string(kubermaticv1.KubevirtCloudProvider),
		Resources: corev1.ResourceList{
			"nvidia.com/GPU": resource.MustParse("1"),
		},
	}}
}

func assertErrorContains(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", expected)
	}
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error %q to contain %q", err, expected)
	}
}
