/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2024 Kubermatic GmbH

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

package defaultapplicationcontroller

import (
	"context"
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	clusterv1alpha1 "k8c.io/machine-controller/pkg/apis/cluster/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	testScheme           = fake.NewScheme()
	applicationNamespace = appskubermaticv1.AppNamespaceSpec{
		Name:   "release-namespace",
		Create: true,
	}
)

const (
	defaultValue = "not-empty:\n  value"
	appVersion   = "v1.2.0"
)

func init() {
	utilruntime.Must(clusterv1alpha1.AddToScheme(testScheme))
}

func TestGetAppNamespace(t *testing.T) {
	testCases := []struct {
		name                        string
		application                 appskubermaticv1.ApplicationDefinition
		defaultApplicationNamespace string
		validate                    func(applications appskubermaticv1.ApplicationDefinition) bool
	}{
		{
			name:        "scenario 1: application namespace should be set to default value when a default value is configured",
			application: *genApplicationDefinition("applicationName", "namespace", "v1.0.0", "", true, false, defaultValue, nil, &applicationNamespace),
			validate: func(application appskubermaticv1.ApplicationDefinition) bool {
				namespace := GetAppNamespace(context.Background(), application)
				return namespace.Name == applicationNamespace.Name
			},
		},
		{
			name:        "scenario 2: application namespace should be set to application name when no default value is configured",
			application: *genApplicationDefinition("applicationName", "namespace", "v1.0.0", "", true, false, defaultValue, nil, nil),
			validate: func(application appskubermaticv1.ApplicationDefinition) bool {
				namespace := GetAppNamespace(context.Background(), application)
				return namespace.Name == application.Name
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// validate the result
			if !test.validate(test.application) {
				t.Fatalf("Test failed for: %v with default appnamespace value %v", test.name, test.defaultApplicationNamespace)
			}
		})
	}
}

func genApplicationDefinition(name, namespace, defaultVersion, defaultDatacenterName string, defaultApp, enforced bool, defaultValues string, defaultRawValues *runtime.RawExtension, defaultNamespace *appskubermaticv1.AppNamespaceSpec) *appskubermaticv1.ApplicationDefinition {
	annotations := map[string]string{}
	selector := appskubermaticv1.DefaultingSelector{}
	if defaultDatacenterName != "" {
		selector.Datacenters = []string{defaultDatacenterName}
	}

	return &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Description:      "Test application definition",
			Method:           appskubermaticv1.HelmTemplateMethod,
			DefaultNamespace: defaultNamespace,
			Versions: []appskubermaticv1.ApplicationVersion{
				{
					Version: "v1.0.0",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "https://charts.example.com/test-chart",
								ChartName:    name,
								ChartVersion: "1.0.0",
							},
						},
					},
				},
				{
					Version: appVersion,
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "https://charts.example.com/test-chart",
								ChartName:    name,
								ChartVersion: appVersion,
							},
						},
					},
				},
				{
					Version: "v1.0.3",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "https://charts.example.com/test-chart",
								ChartName:    name,
								ChartVersion: "1.0.3",
							},
						},
					},
				},
			},
			DefaultValuesBlock: defaultValues,
			DefaultValues:      defaultRawValues,
			DefaultVersion:     defaultVersion,
			Default:            defaultApp,
			Enforced:           enforced,
			Selector:           selector,
		},
	}
}
