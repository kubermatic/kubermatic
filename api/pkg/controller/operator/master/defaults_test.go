package master

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/zapr"

	"go.uber.org/zap"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestDefaultingConfigurations(t *testing.T) {
	clientID := "foobar"

	tests := []struct {
		name     string
		input    *operatorv1alpha1.KubermaticConfiguration
		validate func(c *operatorv1alpha1.KubermaticConfiguration) error
	}{
		{
			name: "Auth fields are defaulted",
			input: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					Domain: "example.com",
					Auth: operatorv1alpha1.KubermaticAuthConfiguration{
						ClientID: clientID,
					},
				},
			},
			validate: func(c *operatorv1alpha1.KubermaticConfiguration) error {
				expected := fmt.Sprintf("%sIssuer", clientID)

				if c.Spec.Auth.IssuerClientID != expected {
					return fmt.Errorf("expected IssuerClientID %s, but got '%s'", expected, c.Spec.Auth.IssuerClientID)
				}

				return nil
			},
		},
	}

	rawLog := zap.NewNop()
	log := rawLog.Sugar()

	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog).WithName("controller_runtime"))
	if err := operatorv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("Failed to register types in Scheme: %v", err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := ctrlruntimefake.NewFakeClient(test.input)

			reconciler := Reconciler{
				Client:   client,
				recorder: record.NewFakeRecorder(10),
				log:      log,
				ctx:      context.Background(),
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      test.input.GetName(),
				Namespace: test.input.GetNamespace(),
			}}

			if _, err := reconciler.Reconcile(request); err != nil {
				t.Fatalf("Reconcile returned an error while none was expected: %v", err)
			}

			key, err := controllerruntimeclient.ObjectKeyFromObject(test.input)
			if err != nil {
				t.Fatalf("Failed to generate a ObjectKey: %v", err)
			}

			defaultedConfig := &operatorv1alpha1.KubermaticConfiguration{}
			if err := client.Get(context.Background(), key, defaultedConfig); err != nil {
				t.Fatalf("Failed to get the KubermaticConfiguration from the client: %v", err)
			}

			if err := test.validate(defaultedConfig); err != nil {
				t.Fatalf("Resulting configuration is not valid: %v", err)
			}
		})
	}
}
