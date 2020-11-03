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

package reconciling

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-test/deep"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	controllerruntimefake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
)

func TestDefaultPodSpec(t *testing.T) {
	defaultProcMountType := corev1.DefaultProcMount
	otherProcMountType := corev1.UnmaskedProcMount

	tests := []struct {
		name           string
		oldObject      corev1.PodSpec
		newObject      corev1.PodSpec
		expectedObject corev1.PodSpec
	}{
		{
			name: "Default values for termination message and pull policies are added",
			newObject: corev1.PodSpec{
				Containers: []corev1.Container{
					{},
				},
			},
			expectedObject: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						ImagePullPolicy:          corev1.PullIfNotPresent,
						TerminationMessagePath:   corev1.TerminationMessagePathDefault,
						TerminationMessagePolicy: corev1.TerminationMessageReadFile,
					},
				},
			},
		},
		{
			name: "The new version of objects is prioritized",
			oldObject: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						ImagePullPolicy:          corev1.PullAlways,
						TerminationMessagePath:   "/dev/old",
						TerminationMessagePolicy: corev1.TerminationMessageReadFile,
					},
				},
			},
			newObject: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						ImagePullPolicy:          corev1.PullNever,
						TerminationMessagePath:   "/dev/new",
						TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
					},
				},
			},
			expectedObject: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						ImagePullPolicy:          corev1.PullNever,
						TerminationMessagePath:   "/dev/new",
						TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
					},
				},
			},
		},
		{
			name: "The procMountType is always retained and cannot be overwritten",
			oldObject: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						SecurityContext: &corev1.SecurityContext{
							ProcMount: &defaultProcMountType,
						},
					},
				},
			},
			newObject: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						SecurityContext: &corev1.SecurityContext{
							ProcMount: &otherProcMountType,
						},
					},
				},
			},
			expectedObject: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						SecurityContext: &corev1.SecurityContext{
							ProcMount: &defaultProcMountType,
						},
						ImagePullPolicy:          corev1.PullIfNotPresent,
						TerminationMessagePath:   corev1.TerminationMessagePathDefault,
						TerminationMessagePolicy: corev1.TerminationMessageReadFile,
					},
				},
			},
		},
		{
			name: "Values are assigned based on container names, not ordering",
			oldObject: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "a",
						SecurityContext: &corev1.SecurityContext{
							ProcMount: &defaultProcMountType,
						},
					},
					{
						Name: "b",
						SecurityContext: &corev1.SecurityContext{
							ProcMount: &otherProcMountType,
						},
					},
				},
			},
			newObject: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "b",
						SecurityContext: &corev1.SecurityContext{},
					},
				},
			},
			expectedObject: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:                     "b",
						ImagePullPolicy:          corev1.PullIfNotPresent,
						TerminationMessagePath:   corev1.TerminationMessagePathDefault,
						TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						SecurityContext: &corev1.SecurityContext{
							ProcMount: &otherProcMountType,
						},
					},
				},
			},
		},
		{
			name: "InitContainers are updated as well",
			oldObject: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name: "a",
						SecurityContext: &corev1.SecurityContext{
							ProcMount: &defaultProcMountType,
						},
					},
					{
						Name: "b",
						SecurityContext: &corev1.SecurityContext{
							ProcMount: &otherProcMountType,
						},
					},
				},
			},
			newObject: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name:            "b",
						SecurityContext: &corev1.SecurityContext{},
					},
				},
			},
			expectedObject: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name:                     "b",
						ImagePullPolicy:          corev1.PullIfNotPresent,
						TerminationMessagePath:   corev1.TerminationMessagePathDefault,
						TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						SecurityContext: &corev1.SecurityContext{
							ProcMount: &otherProcMountType,
						},
					},
				},
			},
		},
		{
			name: "InitContainers and Containers are not mixed up",
			oldObject: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name: "a",
						SecurityContext: &corev1.SecurityContext{
							ProcMount: &otherProcMountType,
						},
					},
				},
				Containers: []corev1.Container{
					{
						Name:            "a",
						SecurityContext: &corev1.SecurityContext{},
					},
				},
			},
			newObject: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name:            "a",
						SecurityContext: &corev1.SecurityContext{},
					},
				},
				Containers: []corev1.Container{
					{
						Name:            "a",
						SecurityContext: &corev1.SecurityContext{},
					},
				},
			},
			expectedObject: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name:                     "a",
						ImagePullPolicy:          corev1.PullIfNotPresent,
						TerminationMessagePath:   corev1.TerminationMessagePathDefault,
						TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						SecurityContext: &corev1.SecurityContext{
							ProcMount: &otherProcMountType,
						},
					},
				},
				Containers: []corev1.Container{
					{
						Name:                     "a",
						ImagePullPolicy:          corev1.PullIfNotPresent,
						TerminationMessagePath:   corev1.TerminationMessagePathDefault,
						TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						SecurityContext:          &corev1.SecurityContext{},
					},
				},
			},
		},
		{
			name: "Default mode for secret volume gets defaulted",
			newObject: corev1.PodSpec{
				Volumes: []corev1.Volume{{
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{},
					},
				},
				}},
			expectedObject: corev1.PodSpec{
				Volumes: []corev1.Volume{{
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: utilpointer.Int32Ptr(0644),
						},
					},
				}},
			},
		},
		{
			name: "Default mode for secret volume doesn't get overwritten",
			newObject: corev1.PodSpec{
				Volumes: []corev1.Volume{{
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: utilpointer.Int32Ptr(0600),
						},
					},
				},
				}},
			expectedObject: corev1.PodSpec{
				Volumes: []corev1.Volume{{
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: utilpointer.Int32Ptr(0600),
						},
					},
				}},
			},
		},
		{
			name: "Default mode for configmap volume gets defaulted",
			newObject: corev1.PodSpec{
				Volumes: []corev1.Volume{{
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{},
					},
				},
				}},
			expectedObject: corev1.PodSpec{
				Volumes: []corev1.Volume{{
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							DefaultMode: utilpointer.Int32Ptr(0644),
						},
					},
				}},
			},
		},
		{
			name: "Default mode for configmap volume doesn't get overwritten",
			newObject: corev1.PodSpec{
				Volumes: []corev1.Volume{{
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							DefaultMode: utilpointer.Int32Ptr(0600),
						},
					},
				},
				}},
			expectedObject: corev1.PodSpec{
				Volumes: []corev1.Volume{{
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							DefaultMode: utilpointer.Int32Ptr(0600),
						},
					},
				}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := DefaultPodSpec(test.oldObject, test.newObject)
			if err != nil {
				t.Errorf("DefaultPodSpec returned an unexpected error: %v", err)
			}

			if diff := deep.Equal(result, test.expectedObject); diff != nil {
				t.Errorf("The PodSpec from the client does not match the expected PodSpec. Diff: \n%v", diff)
			}
		})
	}
}

func TestDefaultDeployment(t *testing.T) {
	const (
		testNamespace    = "default"
		testResourceName = "test"
	)

	creators := []NamedDeploymentCreatorGetter{
		func() (string, DeploymentCreator) {
			return testResourceName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
				return d, nil
			}
		},
	}

	existingObject := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testResourceName,
			Namespace: testNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
		},
	}

	expectedObject := &appsv1.Deployment{
		ObjectMeta: existingObject.ObjectMeta,
		Spec: appsv1.DeploymentSpec{
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 1,
					},
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 0,
					},
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							ImagePullPolicy:          corev1.PullIfNotPresent,
							TerminationMessagePath:   corev1.TerminationMessagePathDefault,
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						},
					},
				},
			},
		},
	}

	client := controllerruntimefake.NewFakeClient(existingObject)
	if err := ReconcileDeployments(context.Background(), creators, testNamespace, client); err != nil {
		t.Errorf("EnsureObject returned an error while none was expected: %v", err)
	}

	key, err := controllerruntimeclient.ObjectKeyFromObject(expectedObject)
	if err != nil {
		t.Fatalf("Failed to generate a ObjectKey for the expected object: %v", err)
	}

	actualDeployment := &appsv1.Deployment{}
	if err := client.Get(context.Background(), key, actualDeployment); err != nil {
		t.Fatalf("Failed to get the Deployment from the client: %v", err)
	}

	// wipe formatting differences
	actualDeployment.TypeMeta = metav1.TypeMeta{}
	actualDeployment.ResourceVersion = ""

	if diff := deep.Equal(actualDeployment, expectedObject); diff != nil {
		t.Errorf("The Deployment from the client does not match the expected Deployment. Diff: \n%v", diff)
	}
}

func TestDefaultStatefulSet(t *testing.T) {
	const (
		testNamespace    = "default"
		testResourceName = "test"
	)

	creators := []NamedStatefulSetCreatorGetter{
		func() (string, StatefulSetCreator) {
			return testResourceName, func(d *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
				return d, nil
			}
		},
	}

	existingObject := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testResourceName,
			Namespace: testNamespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
		},
	}

	expectedObject := &appsv1.StatefulSet{
		ObjectMeta: existingObject.ObjectMeta,
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							ImagePullPolicy:          corev1.PullIfNotPresent,
							TerminationMessagePath:   corev1.TerminationMessagePathDefault,
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						},
					},
				},
			},
		},
	}

	client := controllerruntimefake.NewFakeClient(existingObject)
	if err := ReconcileStatefulSets(context.Background(), creators, testNamespace, client); err != nil {
		t.Errorf("EnsureObject returned an error while none was expected: %v", err)
	}

	key, err := controllerruntimeclient.ObjectKeyFromObject(expectedObject)
	if err != nil {
		t.Fatalf("Failed to generate a ObjectKey for the expected object: %v", err)
	}

	actualStatefulSet := &appsv1.StatefulSet{}
	if err := client.Get(context.Background(), key, actualStatefulSet); err != nil {
		t.Fatalf("Failed to get the StatefulSet from the client: %v", err)
	}

	// wipe formatting differences
	actualStatefulSet.TypeMeta = metav1.TypeMeta{}
	actualStatefulSet.ResourceVersion = ""

	if diff := deep.Equal(actualStatefulSet, expectedObject); diff != nil {
		t.Errorf("The StatefulSet from the client does not match the expected StatefulSet. Diff: \n%v", diff)
	}
}

func TestDefaultDaemonSet(t *testing.T) {
	const (
		testNamespace    = "default"
		testResourceName = "test"
	)

	creators := []NamedDaemonSetCreatorGetter{
		func() (string, DaemonSetCreator) {
			return testResourceName, func(d *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
				return d, nil
			}
		},
	}

	existingObject := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testResourceName,
			Namespace: testNamespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{},
					},
				},
			},
		},
	}

	expectedObject := &appsv1.DaemonSet{
		ObjectMeta: existingObject.ObjectMeta,
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							ImagePullPolicy:          corev1.PullIfNotPresent,
							TerminationMessagePath:   corev1.TerminationMessagePathDefault,
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						},
					},
				},
			},
		},
	}

	client := controllerruntimefake.NewFakeClient(existingObject)
	if err := ReconcileDaemonSets(context.Background(), creators, testNamespace, client); err != nil {
		t.Errorf("EnsureObject returned an error while none was expected: %v", err)
	}

	key, err := controllerruntimeclient.ObjectKeyFromObject(expectedObject)
	if err != nil {
		t.Fatalf("Failed to generate a ObjectKey for the expected object: %v", err)
	}

	actualDaemonSet := &appsv1.DaemonSet{}
	if err := client.Get(context.Background(), key, actualDaemonSet); err != nil {
		t.Fatalf("Failed to get the DaemonSet from the client: %v", err)
	}

	// wipe formatting differences
	actualDaemonSet.TypeMeta = metav1.TypeMeta{}
	actualDaemonSet.ResourceVersion = ""

	if diff := deep.Equal(actualDaemonSet, expectedObject); diff != nil {
		t.Errorf("The DaemonSet from the client does not match the expected DaemonSet. Diff: \n%v", diff)
	}
}

func TestDefaultCronJob(t *testing.T) {
	const (
		testNamespace    = "default"
		testResourceName = "test"
	)

	creators := []NamedCronJobCreatorGetter{
		func() (string, CronJobCreator) {
			return testResourceName, func(d *batchv1beta1.CronJob) (*batchv1beta1.CronJob, error) {
				return d, nil
			}
		},
	}

	existingObject := &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testResourceName,
			Namespace: testNamespace,
		},
		Spec: batchv1beta1.CronJobSpec{
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{},
							},
						},
					},
				},
			},
		},
	}

	expectedObject := &batchv1beta1.CronJob{
		ObjectMeta: existingObject.ObjectMeta,
		Spec: batchv1beta1.CronJobSpec{
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									ImagePullPolicy:          corev1.PullIfNotPresent,
									TerminationMessagePath:   corev1.TerminationMessagePathDefault,
									TerminationMessagePolicy: corev1.TerminationMessageReadFile,
								},
							},
						},
					},
				},
			},
		},
	}

	client := controllerruntimefake.NewFakeClient(existingObject)
	if err := ReconcileCronJobs(context.Background(), creators, testNamespace, client); err != nil {
		t.Errorf("EnsureObject returned an error while none was expected: %v", err)
	}

	key, err := controllerruntimeclient.ObjectKeyFromObject(expectedObject)
	if err != nil {
		t.Fatalf("Failed to generate a ObjectKey for the expected object: %v", err)
	}

	actualCronJob := &batchv1beta1.CronJob{}
	if err := client.Get(context.Background(), key, actualCronJob); err != nil {
		t.Fatalf("Failed to get the CronJob from the client: %v", err)
	}

	// wipe formatting differences
	actualCronJob.TypeMeta = metav1.TypeMeta{}
	actualCronJob.ResourceVersion = ""

	if diff := deep.Equal(actualCronJob, expectedObject); diff != nil {
		t.Errorf("The CronJob from the client does not match the expected CronJob. Diff: \n%v", diff)
	}
}

func TestDeploymentStrategyDefaulting(t *testing.T) {
	testCases := []struct {
		name   string
		in     *appsv1.Deployment
		verify func(*appsv1.Deployment) error
	}{
		{
			name: "Strategy and strategysettings get defaulted",
			in:   &appsv1.Deployment{},
			verify: func(d *appsv1.Deployment) error {
				if d.Spec.Strategy.Type != appsv1.RollingUpdateDeploymentStrategyType {
					return fmt.Errorf("expected strategy to be %q, was %q",
						appsv1.RollingUpdateDeploymentStrategyType, d.Spec.Strategy.Type)
				}
				if d.Spec.Strategy.RollingUpdate == nil {
					return errors.New("expected .Spec.Strategy.RollingUpdate to get defaulted, was nil")
				}
				if d.Spec.Strategy.RollingUpdate.MaxSurge == nil {
					return errors.New("expected .Spec.Strategy.RollingUpdate.MaxSurge to get dafaulted, was nil")
				}
				if d.Spec.Strategy.RollingUpdate.MaxSurge.IntVal != 1 {
					return fmt.Errorf("expected .Spec.Strategy.RollingUpdate.MaxSurge to be 1, was %d",
						d.Spec.Strategy.RollingUpdate.MaxSurge.IntVal)
				}
				if d.Spec.Strategy.RollingUpdate.MaxUnavailable == nil {
					return errors.New("expected .Spec.Strategy.RollingUpdate.MaxUnavailable to get defaulted, was nil")
				}
				if d.Spec.Strategy.RollingUpdate.MaxUnavailable.IntVal != 0 {
					return fmt.Errorf("expected .Spec.Strategy.RollingUpdate.MaxUnavailable to be 0, was %d",
						d.Spec.Strategy.RollingUpdate.MaxUnavailable.IntVal)
				}
				return nil
			},
		},
		{
			name: "Strategysettings dont get defaulted when already set",
			in: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Strategy: appsv1.DeploymentStrategy{
						RollingUpdate: &appsv1.RollingUpdateDeployment{
							MaxSurge: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 10,
							},
							MaxUnavailable: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 2,
							},
						},
					},
				},
			},
			verify: func(d *appsv1.Deployment) error {
				if d.Spec.Strategy.Type != appsv1.RollingUpdateDeploymentStrategyType {
					return fmt.Errorf("expected strategy to get defaulted to %q, was %q",
						appsv1.RollingUpdateDeploymentStrategyType, d.Spec.Strategy)
				}
				if d.Spec.Strategy.RollingUpdate.MaxSurge.IntVal != 10 {
					return fmt.Errorf("expected .Spec.Strategy.RollingUpdate.MaxSurge.IntVal to be 10, was %d",
						d.Spec.Strategy.RollingUpdate.MaxSurge.IntVal)
				}
				if d.Spec.Strategy.RollingUpdate.MaxUnavailable.IntVal != 2 {
					return fmt.Errorf("expected .Spec.Strategy.RollingUpdate.MaxUnavailable to be 2, was %d",
						d.Spec.Strategy.RollingUpdate.MaxUnavailable.IntVal)
				}
				return nil
			},
		},
		{
			name: "Both strategy and strategysettings dont get defaulted when strategy is set",
			in: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Strategy: appsv1.DeploymentStrategy{
						Type: appsv1.RecreateDeploymentStrategyType,
					},
				},
			},
			verify: func(d *appsv1.Deployment) error {
				if d.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType {
					return fmt.Errorf("expected strategy to remain %q, was updated to %q",
						appsv1.RecreateDeploymentStrategyType, d.Spec.Strategy)
				}
				if d.Spec.Strategy.RollingUpdate != nil {
					return errors.New("expected .Spec.Strategy.RollingUpdate to remain nil, got set")
				}
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			creator := func(_ *appsv1.Deployment) (*appsv1.Deployment, error) {
				return tc.in, nil
			}
			creator = DefaultDeployment(creator)
			deployment, err := creator(&appsv1.Deployment{})
			if err != nil {
				t.Fatalf("error when calling creator: %v", err)
			}

			if err := tc.verify(deployment); err != nil {
				t.Fatal(err)
			}
		})
	}

}

func TestImagePullSecretsWrapper(t *testing.T) {
	tests := []struct {
		name            string
		secretNames     []string
		inputObj        runtime.Object
		wantSecretNames []string
		wantErr         bool
	}{
		{
			name:            "No secret name provided",
			secretNames:     []string{},
			inputObj:        &appsv1.Deployment{},
			wantSecretNames: []string{},
		},
		{
			name:            "Secret name provided",
			secretNames:     []string{"secret"},
			inputObj:        &appsv1.Deployment{},
			wantSecretNames: []string{"secret"},
		},
		{
			name:            "Secret names provided",
			secretNames:     []string{"secret_1", "secret_2"},
			inputObj:        &appsv1.Deployment{},
			wantSecretNames: []string{"secret_1", "secret_2"},
		},
		{
			name:        "Secret already present",
			secretNames: []string{"secret_1", "secret_2"},
			inputObj: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							ImagePullSecrets: []corev1.LocalObjectReference{
								{Name: "secret_1"},
								{Name: "secret_3"},
							},
						},
					},
				},
			},
			wantSecretNames: []string{"secret_1", "secret_2", "secret_3"},
		},
		{
			name:        "Unsupported object type",
			secretNames: []string{"secret"},
			inputObj:    &appsv1.StatefulSet{TypeMeta: metav1.TypeMeta{Kind: "StatefulSet", APIVersion: "apps/v1"}},
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ImagePullSecretsWrapper(tt.secretNames...)
			create := got(identityCreator)
			_, err := create(tt.inputObj)
			if (err != nil) != tt.wantErr {
				t.Fatalf("wanted error = %v, but got %v", tt.wantErr, err)
			}
			if !tt.wantErr {
				if d, ok := tt.inputObj.(*appsv1.Deployment); ok {
					actualSecretNames := sets.NewString()
					for _, ips := range d.Spec.Template.Spec.ImagePullSecrets {
						actualSecretNames.Insert(ips.Name)
					}
					if len(d.Spec.Template.Spec.ImagePullSecrets) != len(tt.wantSecretNames) || !actualSecretNames.HasAll(tt.wantSecretNames...) {
						t.Errorf("actual and expected image pull secret names do not match. expected: %v actual: %v", tt.wantSecretNames, actualSecretNames.List())
					}
				} else {
					t.Fatal("this is an unexpected condition for this test that today only supports Deployments, if support for other resource types has been added please update this test accordingly")
				}
			}
		})
	}
}

// identityCreator is an ObjectModifier that returns the input object
// untouched.
// TODO(irozzo) May be useful to move this in a test package?
func identityCreator(obj runtime.Object) (runtime.Object, error) {
	return obj, nil
}
