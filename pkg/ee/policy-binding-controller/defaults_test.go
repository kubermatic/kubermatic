//go:build ee

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
	"encoding/json"
	"testing"
)

func TestPolicySpecWithDefaultsPreservesLargeIntegers(t *testing.T) {
	// Decoding arbitrary policy content through float64 would round this value
	// and silently change which resources the policy accepts.
	raw := []byte(`{
		"rules": [{
			"name": "require-exact-counter",
			"match": {"any": [{"resources": {"kinds": ["example.io/v1/Counter"]}}]},
			"validate": {"pattern": {"spec": {"counter": 9007199254740993}}}
		}]
	}`)
	for _, kind := range []string{"ClusterPolicy", "Policy"} {
		t.Run(kind, func(t *testing.T) {
			spec, err := policySpecWithDefaults(raw, kind)
			if err != nil {
				t.Fatalf("apply policy defaults: %v", err)
			}
			if len(spec.Rules) != 1 || spec.Rules[0].Validation == nil || spec.Rules[0].Validation.RawPattern == nil {
				t.Fatal("defaulted policy does not contain the validation pattern")
			}
			var pattern struct {
				Spec struct {
					Counter int64 `json:"counter"`
				} `json:"spec"`
			}
			if err := json.Unmarshal(spec.Rules[0].Validation.RawPattern.Raw, &pattern); err != nil {
				t.Fatalf("decode defaulted validation pattern: %v", err)
			}
			if pattern.Spec.Counter != 9007199254740993 {
				t.Errorf("validation pattern counter changed to %d, want 9007199254740993", pattern.Spec.Counter)
			}
		})
	}
}

func TestPolicySpecWithDefaultsRejectsNull(t *testing.T) {
	for _, kind := range []string{"ClusterPolicy", "Policy"} {
		t.Run(kind, func(t *testing.T) {
			if _, err := policySpecWithDefaults([]byte(`null`), kind); err == nil {
				t.Fatal("expected an error for a null policy spec")
			}
		})
	}
}

func TestPolicySpecWithDefaultsNestedFields(t *testing.T) {
	testCases := []struct {
		name       string
		raw        string
		wantBool   bool
		wantMethod string
	}{
		{
			name: "omitted settings receive defaults inside rules and context entries",
			raw: `{
				"rules": [{
					"name": "require-label",
					"context": [{"name": "namespaces", "apiCall": {"urlPath": "/api/v1/namespaces"}}],
					"match": {"any": [{"resources": {"kinds": ["ConfigMap"]}}]},
					"validate": {"pattern": {"metadata": {"labels": {"app": "required"}}}}
				}]
			}`,
			wantBool:   true,
			wantMethod: "GET",
		},
		{
			name: "explicit settings override defaults",
			raw: `{
				"admission": false,
				"background": false,
				"rules": [{
					"name": "require-label",
					"skipBackgroundRequests": false,
					"context": [{"name": "namespaces", "apiCall": {"urlPath": "/api/v1/namespaces", "method": "POST"}}],
					"match": {"any": [{"resources": {"kinds": ["ConfigMap"]}}]},
					"validate": {"allowExistingViolations": false, "pattern": {"metadata": {"labels": {"app": "required"}}}}
				}]
			}`,
			wantBool:   false,
			wantMethod: "POST",
		},
	}
	for _, kind := range []string{"ClusterPolicy", "Policy"} {
		t.Run(kind, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					spec, err := policySpecWithDefaults([]byte(tc.raw), kind)
					if err != nil {
						t.Fatalf("apply policy defaults: %v", err)
					}
					if len(spec.Rules) != 1 || spec.Rules[0].Validation == nil {
						t.Fatal("defaulted policy does not contain the validation rule")
					}
					rule := spec.Rules[0]
					for name, actual := range map[string]*bool{
						"admission":               spec.Admission,
						"background":              spec.Background,
						"skipBackgroundRequests":  rule.SkipBackgroundRequests,
						"allowExistingViolations": rule.Validation.AllowExistingViolations,
					} {
						if actual == nil {
							t.Errorf("%s is unset, want %t", name, tc.wantBool)
						} else if *actual != tc.wantBool {
							t.Errorf("%s = %t, want %t", name, *actual, tc.wantBool)
						}
					}
					if len(rule.Context) != 1 || rule.Context[0].APICall == nil {
						t.Fatal("defaulted policy does not contain the API context entry")
					}
					if actual := string(rule.Context[0].APICall.Method); actual != tc.wantMethod {
						t.Errorf("context API method = %q, want %q", actual, tc.wantMethod)
					}
				})
			}
		})
	}
}
