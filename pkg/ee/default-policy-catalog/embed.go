/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2025 Kubermatic GmbH

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

package defaultpolicycatalog

import (
	"embed"
	"fmt"
	"io"
	"io/fs"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

//go:embed policies
var f embed.FS

func GetPolicyTemplates() ([]kubermaticv1.PolicyTemplate, error) {
	dirname := "policies"
	templates := []kubermaticv1.PolicyTemplate{}
	entries, err := f.ReadDir(dirname)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			// Open the ClusterPolicy file
			file, err := f.Open(dirname + "/" + entry.Name())
			if err != nil {
				return nil, err
			}

			// Convert the ClusterPolicy to PolicyTemplate
			template, err := convertClusterPolicyToPolicyTemplate(file)
			if err != nil {
				return nil, fmt.Errorf("failed to convert %s: %w", entry.Name(), err)
			}

			templates = append(templates, template)
		}
	}
	return templates, nil
}

// convertClusterPolicyToPolicyTemplate converts a Kyverno ClusterPolicy to a Kubermatic PolicyTemplate.
func convertClusterPolicyToPolicyTemplate(file fs.File) (kubermaticv1.PolicyTemplate, error) {
	content, err := io.ReadAll(file)
	if err != nil {
		return kubermaticv1.PolicyTemplate{}, err
	}

	// Unmarshal the ClusterPolicy into an unstructured object
	var obj unstructured.Unstructured
	if err := yaml.Unmarshal(content, &obj); err != nil {
		return kubermaticv1.PolicyTemplate{}, err
	}

	// Extract metadata from annotations
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	title := annotations["policies.kyverno.io/title"]
	category := annotations["policies.kyverno.io/category"]
	description := annotations["policies.kyverno.io/description"]
	severity := annotations["policies.kyverno.io/severity"]

	// Get the policy spec
	policySpec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return kubermaticv1.PolicyTemplate{}, err
	}
	if !found {
		return kubermaticv1.PolicyTemplate{}, fmt.Errorf("spec not found in ClusterPolicy")
	}

	policySpecObj := &unstructured.Unstructured{
		Object: policySpec,
	}

	// Create the PolicyTemplate
	policyTemplate := kubermaticv1.PolicyTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubermatic.k8c.io/v1",
			Kind:       "PolicyTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName(),
		},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Title:       title,
			Description: description,
			Category:    category,
			Severity:    severity,
			Visibility:  "Global",
			PolicySpec:  runtime.RawExtension{Object: policySpecObj},
		},
	}

	return policyTemplate, nil
}
