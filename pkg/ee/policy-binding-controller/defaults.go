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
	"fmt"
	"sync"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"

	userclusterresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	structuraldefaulting "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/defaulting"
	"k8s.io/apimachinery/pkg/util/json"
)

var policySpecSchemas = sync.OnceValues(loadPolicySpecSchemas)

// policySpecWithDefaults applies the defaults from the CRDs installed by KKP.
// Without these defaults, the reconciler attempts to remove server-defaulted
// fields on every pass. The API server restores them, leaving the object
// unchanged, and the reconciler times out waiting for an update in its cache.
func policySpecWithDefaults(raw []byte, kind string) (kyvernov1.Spec, error) {
	var spec kyvernov1.Spec
	var object map[string]interface{}
	if err := json.Unmarshal(raw, &object); err != nil {
		return spec, err
	}
	if object == nil {
		return spec, fmt.Errorf("policySpec must be a JSON object")
	}
	schemas, err := policySpecSchemas()
	if err != nil {
		return spec, err
	}
	schema, ok := schemas[kind]
	if !ok {
		return spec, fmt.Errorf("no Kyverno policy spec schema for %s", kind)
	}
	// Start from the template, not the existing policy, so removed settings
	// revert to their defaults instead of retaining previously configured values.
	structuraldefaulting.Default(object, schema)
	defaulted, err := json.Marshal(object)
	if err != nil {
		return spec, err
	}
	if err := json.Unmarshal(defaulted, &spec); err != nil {
		return spec, err
	}
	return spec, nil
}

func loadPolicySpecSchemas() (map[string]*structuralschema.Structural, error) {
	crds, err := userclusterresources.KyvernoCRDs()
	if err != nil {
		return nil, fmt.Errorf("load Kyverno CRDs for policy defaults: %w", err)
	}
	schemas := make(map[string]*structuralschema.Structural, 2)
	for _, crd := range crds {
		kind := crd.Spec.Names.Kind
		if kind != "Policy" && kind != "ClusterPolicy" {
			continue
		}
		for _, version := range crd.Spec.Versions {
			if version.Name != kyvernov1.GroupVersion.Version || version.Schema == nil || version.Schema.OpenAPIV3Schema == nil {
				continue
			}
			specSchema, ok := version.Schema.OpenAPIV3Schema.Properties["spec"]
			if !ok {
				return nil, fmt.Errorf("Kyverno %s CRD has no spec schema", kind)
			}
			internalSchema := &apiextensions.JSONSchemaProps{}
			if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(&specSchema, internalSchema, nil); err != nil {
				return nil, fmt.Errorf("convert Kyverno %s spec schema: %w", kind, err)
			}
			schema, err := structuralschema.NewStructural(internalSchema)
			if err != nil {
				return nil, fmt.Errorf("build Kyverno %s structural schema: %w", kind, err)
			}
			schemas[kind] = schema
			break
		}
	}
	for _, kind := range []string{"ClusterPolicy", "Policy"} {
		if _, ok := schemas[kind]; !ok {
			return nil, fmt.Errorf("no Kyverno %s spec schema for version %s", kind, kyvernov1.GroupVersion.Version)
		}
	}
	return schemas, nil
}
