package clusterdeletion

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
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
			name:    "Dont delete pod without PV",
			objects: []runtime.Object{getPod("", "", false)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewFakeClientWithScheme(scheme.Scheme, tc.objects...)
			d := &Deletion{}
			ctx := context.Background()

			if err := d.cleanupPVCUsingPods(ctx, client); (err != nil) != tc.errExpected {
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
