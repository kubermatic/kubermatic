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
	"encoding/json"
	"fmt"
	"io"
	"io/fs"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			closeErr := file.Close()

			if err != nil {
				return nil, fmt.Errorf("failed to convert %s: %w", entry.Name(), err)
			}
			if closeErr != nil {
				return nil, fmt.Errorf("failed to close %s: %w", entry.Name(), closeErr)
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

	// Unmarshal the ClusterPolicy
	var clusterPolicy kyvernov1.ClusterPolicy
	if err := yaml.UnmarshalStrict(content, &clusterPolicy); err != nil {
		return kubermaticv1.PolicyTemplate{}, fmt.Errorf("failed to convert Kyverno ClusterPolicy to Kubermatic PolicyTemplate: %w", err)
	}

	specJSON, err := canonicalSpec(&clusterPolicy)
	if err != nil {
		return kubermaticv1.PolicyTemplate{}, err
	}

	// Extract metadata from annotations
	annotations := clusterPolicy.Annotations

	title := annotations["policies.kyverno.io/title"]
	category := annotations["policies.kyverno.io/category"]
	description := annotations["policies.kyverno.io/description"]
	severity := annotations["policies.kyverno.io/severity"]

	// Create a RawExtension with the JSON data
	rawExtension := runtime.RawExtension{
		Raw: specJSON,
	}

	// Create the PolicyTemplate
	policyTemplate := kubermaticv1.PolicyTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubermatic.k8c.io/v1",
			Kind:       "PolicyTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterPolicy.Name,
		},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Title:       title,
			Description: description,
			Category:    category,
			Severity:    severity,
			Visibility:  "Global",
			PolicySpec:  rawExtension,
		},
	}

	return policyTemplate, nil
}

func canonicalSpec(policy *kyvernov1.ClusterPolicy) ([]byte, error) {
	specMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&policy.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to convert policy spec to unstructured: %w", err)
	}

	specJSON, err := json.Marshal(specMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy spec: %w", err)
	}

	return append([]byte(nil), specJSON...), nil
}
