package clusterdeletion

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testNS = "test-ns"

func getPod(ownerRefKind, ownerRefName string, hasPV bool) *corev1.Pod {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      "my-pod",
		},
	}

	if ownerRefKind != "" {
		p.OwnerReferences = []metav1.OwnerReference{{Kind: ownerRefKind, Name: ownerRefName}}
	}

	if hasPV {
		p.Spec.Volumes = []corev1.Volume{
			{
				Name: "my-vol",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{},
				},
			},
		}
	}

	return p
}

func TestCleanUpPVUsingWorkloads(t *testing.T) {
	testCases := []struct {
		name                string
		objects             []runtime.Object
		errExpected         bool
		objDeletionExpected bool
	}{
		{
			name:                "Delete Pod",
			objects:             []runtime.Object{getPod("", "", true)},
			objDeletionExpected: true,
		},
		{
			name: "Delete DaemonSet",
			objects: []runtime.Object{
				getPod("DaemonSet", "my-ds", true),
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS,
						Name:      "my-ds",
					},
				},
			},
			objDeletionExpected: true,
		},
		{
			name: "Delete StatefulSet",
			objects: []runtime.Object{
				getPod("StatefulSet", "my-sst", true),
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS,
						Name:      "my-sst",
					},
				},
			},
			objDeletionExpected: true,
		},
		{
			name: "Delete Deployment",
			objects: []runtime.Object{
				getPod("ReplicaSet", "my-rs", true),
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       testNS,
						Name:            "my-rs",
						OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "my-dep"}},
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS,
						Name:      "my-dep",
					},
				},
			},
			objDeletionExpected: true,
		},
		{
			name: "Fail deleting unknown Object",
			objects: []runtime.Object{
				getPod("UnknownKind", "my-obj", true),
			},
			errExpected: true,
		},
		{
			name:    "Dont delete pod without PV",
			objects: []runtime.Object{getPod("", "", false)},
		},
		{
			name:                "No error when owner doesn't exist",
			objects:             []runtime.Object{getPod("ReplicaSet", "my-rs", true)},
			objDeletionExpected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewFakeClient(tc.objects...)
			d := &Deletion{userClusterClient: client}
			ctx := context.Background()

			if _, err := d.cleanupPVUsingWorkloads(ctx); (err == nil) == tc.errExpected {
				t.Fatalf("Expected err=%v, got err=%v", tc.errExpected, err)
			}
			if tc.errExpected {
				return
			}

			for _, object := range tc.objects {
				metav1Object := object.(metav1.Object)
				nn := types.NamespacedName{
					Namespace: metav1Object.GetNamespace(),
					Name:      metav1Object.GetName(),
				}

				err := client.Get(ctx, nn, object.DeepCopyObject())
				if kerrors.IsNotFound(err) != tc.objDeletionExpected {
					t.Errorf("Expected object %q to be deleted=%t", nn.String(), tc.objDeletionExpected)
				}
			}
		})
	}
}
