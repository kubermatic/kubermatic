package operatormaster

import (
	"context"
	"fmt"
	"testing"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type nopEventRecorder struct{}

// These implement the record.EventRecorder interface.

func (n *nopEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
}
func (n *nopEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
}
func (n *nopEventRecorder) PastEventf(object runtime.Object, timestamp metav1.Time, eventtype, reason, messageFmt string, args ...interface{}) {
}
func (n *nopEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
}

func TestDefaultingConfigurations(t *testing.T) {
	tests := []struct {
		name     string
		input    *operatorv1alpha1.KubermaticConfiguration
		validate func(c *operatorv1alpha1.KubermaticConfiguration) error
	}{
		{
			name: "Namespace is defaulted to the configuration's namespace",
			input: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			},
			validate: func(c *operatorv1alpha1.KubermaticConfiguration) error {
				if c.Spec.Namespace != c.Namespace {
					return fmt.Errorf("expected namespace %s, but got '%s'", c.Namespace, c.Spec.Namespace)
				}

				return nil
			},
		},
	}

	rawLog := kubermaticlog.New(true, kubermaticlog.FormatJSON)
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	controllerruntime.SetLogger(nil)

	if err := operatorv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatalw("Failed to register types in Scheme", "error", err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := controllerruntimefake.NewFakeClient(test.input)

			reconciler := Reconciler{
				Client:   client,
				recorder: &nopEventRecorder{},
				log:      log,
				ctx:      context.Background(),
			}

			log.Info("hallo world")

			request := reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      test.input.GetName(),
				Namespace: test.input.GetNamespace(),
			}}

			if _, err := reconciler.Reconcile(request); err != nil {
				t.Errorf("Reconcile returned an error while none was expected: %v", err)
			}

			// key, err := controllerruntimeclient.ObjectKeyFromObject(test.input)
			// if err != nil {
			// 	t.Errorf("Failed to generate a ObjectKey: %v", err)
			// }

			// defaultedConfig := &operatorv1alpha1.KubermaticConfiguration{}
			// if err := client.Get(context.Background(), key, defaultedConfig); err != nil {
			// 	t.Errorf("Failed to get the KubermaticConfiguration from the client: %v", err)
			// }

			// if err := test.validate(defaultedConfig); err != nil {
			// 	t.Errorf("Resulting configuration is not valid: %v", err)
			// }
		})
	}
}
