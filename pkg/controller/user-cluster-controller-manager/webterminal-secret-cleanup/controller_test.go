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

package webterminalsecretcleanup

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"testing"
	"time"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var now = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

func md5Name(email string) string {
	hash := md5.Sum([]byte(email))
	return resources.WebTerminalKubeconfigSecretPrefix + hex.EncodeToString(hash[:])
}

type secretOpt func(*corev1.Secret)

func withLabel() secretOpt {
	return func(s *corev1.Secret) {
		s.Labels = map[string]string{resources.AppLabelKey: resources.WebTerminalAppLabelValue}
	}
}

func withUser(email string) secretOpt {
	return func(s *corev1.Secret) {
		s.Annotations = map[string]string{resources.WebTerminalUserAnnotationKey: email}
	}
}

func withNamespace(namespace string) secretOpt {
	return func(s *corev1.Secret) {
		s.Namespace = namespace
	}
}

func writtenAt(manager string, t time.Time) secretOpt {
	return func(s *corev1.Secret) {
		s.ManagedFields = append(s.ManagedFields, metav1.ManagedFieldsEntry{
			Manager:    manager,
			Operation:  metav1.ManagedFieldsOperationUpdate,
			APIVersion: "v1",
			Time:       &metav1.Time{Time: t},
		})
	}
}

func genSecret(name string, created time.Time, opts ...secretOpt) *corev1.Secret {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         metav1.NamespaceSystem,
			CreationTimestamp: metav1.Time{Time: created},
		},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func genEvictedTerminalPod(name, secretName string) *corev1.Pod {
	pod := genTerminalPod(name, secretName, false)
	pod.Status.Phase = corev1.PodFailed
	pod.Status.Reason = "Evicted"
	return pod
}

func genTerminalPod(name, secretName string, terminating bool) *corev1.Pod {
	return genPod(name, secretName, map[string]string{resources.AppLabelKey: resources.WebTerminalAppLabelValue}, terminating)
}

func genPod(name, secretName string, labels map[string]string, terminating bool) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceSystem,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{{
				Name: "kubeconfig",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{SecretName: secretName},
				},
			}},
		},
	}
	if terminating {
		// the fake client needs a finalizer to accept a deletionTimestamp
		pod.DeletionTimestamp = &metav1.Time{Time: now.Add(-time.Hour)}
		pod.Finalizers = []string{"example.com/stuck"}
	}
	return pod
}

func TestReconcile(t *testing.T) {
	const (
		lowerEmail = "user@example.com"
		mixedEmail = "User.Name@Example.com"
	)

	old := now.Add(-time.Hour)
	young := now.Add(-5 * time.Minute)
	exactlyMinAge := now.Add(-minSecretAge)

	testCases := []struct {
		name          string
		secret        *corev1.Secret
		pods          []ctrlruntimeclient.Object
		paused        bool
		expectDeleted bool
		expectRequeue time.Duration
	}{
		{
			name:          "labelled Secret, old, no pod: deleted",
			secret:        genSecret(md5Name(lowerEmail), old, withLabel(), withUser(lowerEmail)),
			expectDeleted: true,
		},
		{
			name:          "labelled Secret whose name does not match its annotation: deleted",
			secret:        genSecret("kubeconfig-0123456789abcdef0123456789abcdef", old, withLabel(), withUser(lowerEmail)),
			expectDeleted: true,
		},
		{
			name:   "labelled Secret without the kubeconfig- prefix: kept",
			secret: genSecret("my-app", old, withLabel(), withUser(lowerEmail)),
		},
		{
			name:          "unlabelled Secret, name is md5 of lowercase email: deleted",
			secret:        genSecret(md5Name(lowerEmail), old, withUser(lowerEmail)),
			expectDeleted: true,
		},
		{
			name:          "unlabelled Secret, name is md5 of mixed-case email (v2.28 to v2.29.2): deleted",
			secret:        genSecret(md5Name(mixedEmail), old, withUser(mixedEmail)),
			expectDeleted: true,
		},
		{
			name:   "kubeconfig- plus 32 hex, annotation does not match, no label: kept",
			secret: genSecret(md5Name(lowerEmail), old, withUser("someone-else@example.com")),
		},
		{
			name:   "kubeconfig- plus 32 hex, no annotation, no label: kept",
			secret: genSecret(md5Name(lowerEmail), old),
		},
		{
			name:   "kubeconfig-control-not-terminal: kept",
			secret: genSecret("kubeconfig-control-not-terminal", old, withUser(lowerEmail)),
		},
		{
			name:   "matching Secret in another namespace: kept",
			secret: genSecret(md5Name(lowerEmail), old, withLabel(), withUser(lowerEmail), withNamespace("default")),
		},
		{
			name:          "matching Secret younger than 15 minutes: kept and requeued",
			secret:        genSecret(md5Name(lowerEmail), young, withLabel(), withUser(lowerEmail)),
			expectRequeue: 10 * time.Minute,
		},
		{
			name:          "matching Secret exactly 15 minutes old: deleted",
			secret:        genSecret(md5Name(lowerEmail), exactlyMinAge, withLabel(), withUser(lowerEmail)),
			expectDeleted: true,
		},
		{
			name:   "young Secret that is not a web terminal Secret: kept, not requeued",
			secret: genSecret(md5Name(lowerEmail), young, withUser("someone-else@example.com")),
		},
		{
			name:          "matching Secret mounted by a live terminal pod: kept and requeued",
			secret:        genSecret(md5Name(lowerEmail), old, withLabel(), withUser(lowerEmail)),
			pods:          []ctrlruntimeclient.Object{genTerminalPod("webterminal-a", md5Name(lowerEmail), false)},
			expectRequeue: minSecretAge,
		},
		{
			name:          "matching Secret mounted only by a terminal pod stuck in Terminating: deleted",
			secret:        genSecret(md5Name(lowerEmail), old, withLabel(), withUser(lowerEmail)),
			pods:          []ctrlruntimeclient.Object{genTerminalPod("webterminal-a", md5Name(lowerEmail), true)},
			expectDeleted: true,
		},
		{
			name:          "matching Secret mounted only by an evicted terminal pod: deleted",
			secret:        genSecret(md5Name(lowerEmail), old, withLabel(), withUser(lowerEmail)),
			pods:          []ctrlruntimeclient.Object{genEvictedTerminalPod("webterminal-a", md5Name(lowerEmail))},
			expectDeleted: true,
		},
		{
			name:          "Secret mounted only by a pod without the web terminal label: deleted",
			secret:        genSecret(md5Name(lowerEmail), old, withLabel(), withUser(lowerEmail)),
			pods:          []ctrlruntimeclient.Object{genPod("some-app", md5Name(lowerEmail), map[string]string{resources.AppLabelKey: "some-app"}, false)},
			expectDeleted: true,
		},
		{
			name:          "live terminal pod mounts a different Secret: deleted",
			secret:        genSecret(md5Name(lowerEmail), old, withLabel(), withUser(lowerEmail)),
			pods:          []ctrlruntimeclient.Object{genTerminalPod("webterminal-b", md5Name(mixedEmail), false)},
			expectDeleted: true,
		},
		{
			name:          "cluster paused: kept and requeued",
			secret:        genSecret(md5Name(lowerEmail), old, withLabel(), withUser(lowerEmail)),
			paused:        true,
			expectRequeue: minSecretAge,
		},
		{
			name:   "cluster paused, not a web terminal Secret: kept, not requeued",
			secret: genSecret("kubeconfig-control-not-terminal", old, withUser(lowerEmail)),
			paused: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			client := fake.NewClientBuilder().
				WithObjects(tc.secret).
				WithObjects(tc.pods...).
				Build()

			r := &reconciler{
				log:             kubermaticlog.Logger,
				client:          client,
				podReader:       client,
				clusterIsPaused: func(context.Context) (bool, error) { return tc.paused, nil },
				now:             func() time.Time { return now },
			}

			key := types.NamespacedName{Namespace: tc.secret.Namespace, Name: tc.secret.Name}

			result, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			if err != nil {
				t.Fatalf("Reconcile returned an error: %v", err)
			}

			if result.RequeueAfter != tc.expectRequeue {
				t.Errorf("expected RequeueAfter %v, got %v", tc.expectRequeue, result.RequeueAfter)
			}

			err = client.Get(ctx, key, &corev1.Secret{})
			deleted := apierrors.IsNotFound(err)
			if err != nil && !deleted {
				t.Fatalf("failed to get Secret: %v", err)
			}

			if deleted != tc.expectDeleted {
				t.Errorf("expected Secret deleted=%v, got deleted=%v", tc.expectDeleted, deleted)
			}
		})
	}
}

// TestLastWritten does not use the fake client because it drops managedFields.
func TestLastWritten(t *testing.T) {
	created := now.Add(-time.Hour)

	testCases := []struct {
		name     string
		secret   *corev1.Secret
		expected time.Time
	}{
		{
			name:     "no managedFields: creation timestamp",
			secret:   genSecret("kubeconfig-x", created),
			expected: created,
		},
		{
			name:     "managedFields newer than creation: managedFields time",
			secret:   genSecret("kubeconfig-x", created, writtenAt("kubermatic-api", now.Add(-time.Minute))),
			expected: now.Add(-time.Minute),
		},
		{
			name:     "several managedFields entries: the newest one",
			secret:   genSecret("kubeconfig-x", created, writtenAt("kubermatic-api", now.Add(-3*time.Minute)), writtenAt("kubectl", now.Add(-30*time.Minute))),
			expected: now.Add(-3 * time.Minute),
		},
		{
			name: "managedFields entry without time: creation timestamp",
			secret: genSecret("kubeconfig-x", created, func(s *corev1.Secret) {
				s.ManagedFields = []metav1.ManagedFieldsEntry{{Manager: "kubermatic-api"}}
			}),
			expected: created,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := lastWritten(tc.secret); !got.Equal(tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestReconcileSecretNotFound(t *testing.T) {
	r := &reconciler{
		log:             kubermaticlog.Logger,
		client:          fake.NewClientBuilder().Build(),
		podReader:       fake.NewClientBuilder().Build(),
		clusterIsPaused: func(context.Context) (bool, error) { return false, nil },
		now:             func() time.Time { return now },
	}

	request := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: md5Name("gone@example.com")}}
	if _, err := r.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("expected no error for a missing Secret, got: %v", err)
	}
}
