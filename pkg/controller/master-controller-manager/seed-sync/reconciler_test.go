package seedsync

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/zapr"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestReconcilingSeed(t *testing.T) {
	existingSeeds := []runtime.Object{
		&kubermaticv1.Seed{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-seed",
				Namespace: "kubermatic",
				Labels: map[string]string{
					"i-should-be-removed": "please",
				},
				Annotations: map[string]string{
					"i-should-be-removed": "please",
				},
			},
			Spec: kubermaticv1.SeedSpec{
				Country: "Val Verde",
			},
		},
	}
	tests := []struct {
		name          string
		shouldFail    bool
		input         *kubermaticv1.Seed
		existingSeeds []runtime.Object
		validate      func(input, result *kubermaticv1.Seed) error
	}{
		{
			name: "Happy path",
			input: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-seed",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.SeedSpec{
					Country:        "Germany",
					ExposeStrategy: corev1.ServiceTypeNodePort,
				},
			},
			existingSeeds: existingSeeds,
			validate: func(input, result *kubermaticv1.Seed) error {
				if result == nil {
					return errors.New("seed CR should exist in seed cluster, but does not")
				}

				if l := result.Labels[ManagedByLabel]; l != ControllerName {
					return fmt.Errorf("seed CR should have a %s label with '%s', but has label '%s'", ManagedByLabel, ControllerName, l)
				}

				if _, exists := result.Labels["i-should-be-removed"]; exists {
					return errors.New("existing labels should have been removed, but were not")
				}

				if _, exists := result.Annotations["i-should-be-removed"]; exists {
					return errors.New("existing annotations should have been removed, but were not")
				}

				if input.Spec.Country != result.Spec.Country {
					return errors.New("existing spec should have been updated, but was not")
				}

				return nil
			},
		},
		{
			name: "Empty Expose Strategy",
			input: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-seed",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.SeedSpec{
					Country: "Germany",
				},
			},
			existingSeeds: existingSeeds,
			validate: func(input, result *kubermaticv1.Seed) error {
				if result == nil {
					return errors.New("seed CR should exist in seed cluster, but does not")
				}

				if l := result.Labels[ManagedByLabel]; l != ControllerName {
					return fmt.Errorf("seed CR should have a %s label with '%s', but has label '%s'", ManagedByLabel, ControllerName, l)
				}

				if _, exists := result.Labels["i-should-be-removed"]; exists {
					return errors.New("existing labels should have been removed, but were not")
				}

				if _, exists := result.Annotations["i-should-be-removed"]; exists {
					return errors.New("existing annotations should have been removed, but were not")
				}

				if input.Spec.Country != result.Spec.Country {
					return errors.New("existing spec should have been updated, but was not")
				}

				return nil
			},
		},
		{
			name:       "Invalid Expose Strategy",
			shouldFail: true,
			input: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-seed",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.SeedSpec{
					Country:        "Germany",
					ExposeStrategy: corev1.ServiceTypeClusterIP,
				},
			},
			existingSeeds: existingSeeds,
			// actual check is done in the reconciler, nothing to validate here
			validate: nil,
		},
	}

	rawLog := zap.NewNop()
	log := rawLog.Sugar()

	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog).WithName("controller_runtime"))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			masterClient := ctrlruntimefake.NewFakeClient(test.input)
			seedClient := ctrlruntimefake.NewFakeClient(test.existingSeeds...)
			ctx := context.Background()

			reconciler := Reconciler{
				Client:   masterClient,
				recorder: record.NewFakeRecorder(10),
				log:      log,
				ctx:      ctx,
				seedClientGetter: func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
					return seedClient, nil
				},
			}

			err := reconciler.reconcile(test.input, seedClient, log)
			if test.shouldFail && err == nil {
				t.Fatalf("check for %s failed", test.name)
			}
			if !test.shouldFail && err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			key, err := ctrlruntimeclient.ObjectKeyFromObject(test.input)
			if err != nil {
				t.Fatalf("could not create object key for seed: %v", err)
			}

			result := &kubermaticv1.Seed{}
			if err := seedClient.Get(ctx, key, result); err != nil && kerrors.IsNotFound(err) {
				t.Fatalf("could not find seed CR in seed cluster: %v", err)
			}
			if test.validate != nil {
				if err := test.validate(test.input, result); err != nil {
					t.Fatalf("reconciling did not lead to a valid end state: %v", err)
				}
			}
		})
	}
}
