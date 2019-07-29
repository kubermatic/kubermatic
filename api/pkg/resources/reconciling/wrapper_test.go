package reconciling

import (
	"context"
	"testing"

	"github.com/go-test/deep"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	controllerruntimefake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if diff := deep.Equal(actualCronJob, expectedObject); diff != nil {
		t.Errorf("The CronJob from the client does not match the expected CronJob. Diff: \n%v", diff)
	}
}
