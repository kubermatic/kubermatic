//go:build ee && integration

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2026 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package policybindingcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	userclusterresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// TestPolicyBindingCRDDefaultingIntegration exercises the API server defaults
// that fake clients omit. Reapplying an incomplete spec can otherwise result in
// a no-op API update followed by a timeout waiting for a changed cached object.
func TestPolicyBindingCRDDefaultingIntegration(t *testing.T) {
	crds, err := userclusterresources.KyvernoCRDs()
	if err != nil {
		t.Fatalf("load embedded Kyverno CRDs: %v", err)
	}
	env := &envtest.Environment{}
	for i := range crds {
		if crds[i].Name == "clusterpolicies.kyverno.io" || crds[i].Name == "policies.kyverno.io" {
			env.CRDs = append(env.CRDs, &crds[i])
		}
	}
	if len(env.CRDs) != 2 {
		t.Fatalf("expected both Kyverno policy CRDs, got %d", len(env.CRDs))
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start API server: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop API server: %v", err)
		}
	})
	scheme := fake.NewScheme()
	if err := kyvernov1.Install(scheme); err != nil {
		t.Fatalf("register Kyverno resources: %v", err)
	}
	userClient, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("create direct user client: %v", err)
	}

	for _, namespaced := range []bool{false, true} {
		name := "cluster-policy"
		if namespaced {
			name = "namespaced-policy"
		}
		t.Run(name, func(t *testing.T) {
			template := genPolicyTemplate(name, namespaced)
			template.Spec.PolicySpec.Raw = policyDefaultingIntegrationSpec("", "initial")
			binding := genPolicyBinding(name, testClusterNamespace, name)
			if namespaced {
				binding.Spec.KyvernoPolicyNamespace = &kubermaticv1.KyvernoPolicyNamespace{Name: name}
			}
			seedClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(genCluster(testClusterName, true), template, binding).
				WithStatusSubresource(binding).
				Build()
			r := &reconciler{
				seedClient:  seedClient,
				userClient:  userClient,
				log:         zap.NewNop().Sugar(),
				recorder:    &events.FakeRecorder{},
				namespace:   testClusterNamespace,
				clusterName: testClusterName,
				clusterIsPaused: func(context.Context) (bool, error) {
					return false, nil
				},
			}
			request := reconcile.Request{NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(binding)}
			reconcileBinding := func(t *testing.T) {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if _, err := r.Reconcile(ctx, request); err != nil {
					t.Fatalf("reconcile binding: %v", err)
				}
				updated := &kubermaticv1.PolicyBinding{}
				if err := seedClient.Get(ctx, request.NamespacedName, updated); err != nil {
					t.Fatalf("get binding: %v", err)
				}
				if updated.Status.Active == nil || !*updated.Status.Active || !meta.IsStatusConditionTrue(updated.Status.Conditions, string(kubermaticv1.PolicyBindingConditionReady)) {
					t.Fatalf("expected active, ready binding, got status %#v", updated.Status)
				}
			}

			for _, phase := range []struct {
				name      string
				admission string
				label     string
			}{
				{name: "creation", label: "initial"},
				{name: "template-update", admission: `"admission":false,`, label: "updated"},
				{name: "restore-defaults", label: "initial"},
			} {
				if !t.Run(phase.name, func(t *testing.T) {
					ctx := context.Background()
					if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(template), template); err != nil {
						t.Fatalf("get template: %v", err)
					}
					template.Spec.PolicySpec.Raw = policyDefaultingIntegrationSpec(phase.admission, phase.label)
					if err := seedClient.Update(ctx, template); err != nil {
						t.Fatalf("update template: %v", err)
					}
					reconcileBinding(t)
					policy, spec := readPolicyDefaultingIntegration(t, userClient, name, namespaced)
					if spec.Admission == nil || *spec.Admission != (phase.admission == "") {
						t.Fatalf("unexpected admission default: %v", spec.Admission)
					}
					if len(spec.Rules) != 1 || spec.Rules[0].Validation == nil || spec.Rules[0].Validation.RawPattern == nil {
						t.Fatalf("expected one validation rule with a pattern, got %#v", spec.Rules)
					}
					var pattern any
					if err := json.Unmarshal(spec.Rules[0].Validation.RawPattern.Raw, &pattern); err != nil {
						t.Fatalf("decode validation pattern: %v", err)
					}
					wantPattern := map[string]any{"metadata": map[string]any{"labels": map[string]any{"required": phase.label}}}
					if !reflect.DeepEqual(pattern, wantPattern) {
						t.Fatalf("expected template validation pattern %v, got %v", wantPattern, pattern)
					}

					reconcileBinding(t)
					repeated, _ := readPolicyDefaultingIntegration(t, userClient, name, namespaced)
					if repeated.GetResourceVersion() != policy.GetResourceVersion() {
						t.Fatalf("unchanged reconciliation updated policy: resourceVersion %s -> %s", policy.GetResourceVersion(), repeated.GetResourceVersion())
					}
				}) {
					return
				}
			}
		})
	}
}

func policyDefaultingIntegrationSpec(admission, label string) []byte {
	return []byte(fmt.Sprintf(`{%s"background":false,"rules":[{"name":"require-label","match":{"any":[{"resources":{"kinds":["ConfigMap"]}}]},"validate":{"failureAction":"Enforce","pattern":{"metadata":{"labels":{"required":%q}}}}}]}`, admission, label))
}

func readPolicyDefaultingIntegration(t *testing.T, client ctrlruntimeclient.Client, name string, namespaced bool) (ctrlruntimeclient.Object, *kyvernov1.Spec) {
	t.Helper()
	key := ctrlruntimeclient.ObjectKey{Name: name}
	if namespaced {
		key.Namespace = name
		policy := &kyvernov1.Policy{}
		if err := client.Get(context.Background(), key, policy); err != nil {
			t.Fatalf("get namespaced policy: %v", err)
		}
		return policy, &policy.Spec
	}
	policy := &kyvernov1.ClusterPolicy{}
	if err := client.Get(context.Background(), key, policy); err != nil {
		t.Fatalf("get cluster policy: %v", err)
	}
	return policy, &policy.Spec
}
