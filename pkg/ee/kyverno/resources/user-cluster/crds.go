//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0")
                     Copyright © 2025 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package resources

import (
	"embed"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// embeddedFS is an embedded fs that contains Kyverno CRD manifests
//
//go:embed static/*
var embeddedFS embed.FS

// CRDs returns a list of CRDs.
func CRDs() ([]apiextensionsv1.CustomResourceDefinition, error) {
	files, err := embeddedFS.ReadDir("static")
	if err != nil {
		return nil, err
	}

	result := []apiextensionsv1.CustomResourceDefinition{}

	for _, info := range files {
		crd, err := loadCRD(info.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to open CRD: %w", err)
		}
		result = append(result, *crd)
	}

	return result, nil
}

func loadCRD(filename string) (*apiextensionsv1.CustomResourceDefinition, error) {
	f, err := embeddedFS.Open("static/" + filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	crd := &apiextensionsv1.CustomResourceDefinition{}
	dec := yaml.NewYAMLOrJSONDecoder(f, 1024)
	if err := dec.Decode(crd); err != nil {
		return nil, err
	}

	return crd, nil
}

// CRDReconciler returns a reconciler for a CRD.
func CRDReconciler(crd apiextensionsv1.CustomResourceDefinition) reconciling.NamedCustomResourceDefinitionReconcilerFactory {
	return func() (string, reconciling.CustomResourceDefinitionReconciler) {
		return crd.Name, func(target *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			kubernetes.EnsureAnnotations(target, crd.Annotations)
			kubernetes.EnsureLabels(target, crd.Labels)

			target.Spec = crd.Spec

			// reconcile fails if conversion is not set as it's set by default to None
			target.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}

			return target, nil
		}
	}
}
