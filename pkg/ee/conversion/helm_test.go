// +build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2020 Loodse GmbH

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

package conversion

import (
	"fmt"
	"strings"
	"testing"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"
)

func parseYAML(t *testing.T, content string) helmValues {
	values := helmValues{}
	if err := yaml.Unmarshal([]byte(strings.TrimSpace(content)), &values); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	return values
}

func TestParsingReplicas(t *testing.T) {
	defaultReplicas := 42
	testcases := []struct {
		yaml     string
		expected *int32
	}{
		{
			yaml:     `kubermatic: {}`,
			expected: nil,
		},
		{
			yaml:     `kubermatic: { api: { replicas: 0 } }`,
			expected: pointer.Int32Ptr(0),
		},
		{
			yaml:     `kubermatic: { api: { replicas: 7 } }`,
			expected: pointer.Int32Ptr(7),
		},
		{
			yaml:     `kubermatic: { api: { replicas: "7" } }`,
			expected: pointer.Int32Ptr(7),
		},
	}

	for idx, testcase := range testcases {
		values := parseYAML(t, testcase.yaml)

		replicas, err := getReplicas(values.Kubermatic.API.Replicas, defaultReplicas)
		if err != nil {
			t.Errorf("Test case %d failed, failed to determine replicas: %v", idx+1, err)
			continue
		}

		switch {
		case replicas == nil && testcase.expected == nil:
			break
		case replicas == nil:
			t.Errorf("Test case %d failed, expected %v but got %v replicas.", idx+1, *testcase.expected, replicas)
		case testcase.expected == nil:
			t.Errorf("Test case %d failed, expected %v but got %v replicas.", idx+1, testcase.expected, *replicas)
		case *replicas != *testcase.expected:
			t.Errorf("Test case %d failed, expected %v but got %v replicas.", idx+1, *testcase.expected, *replicas)
		}
	}
}

func TestConvertAuth(t *testing.T) {
	testcases := []struct {
		yaml     string
		expected *operatorv1alpha1.KubermaticAuthConfiguration
	}{
		{
			yaml:     `kubermatic: {}`,
			expected: &operatorv1alpha1.KubermaticAuthConfiguration{},
		},
		{
			yaml: `kubermatic: { auth: { clientID: foo } }`,
			expected: &operatorv1alpha1.KubermaticAuthConfiguration{
				ClientID: "foo",
			},
		},
		{
			yaml:     fmt.Sprintf(`kubermatic: { auth: { clientID: "%s" } }`, common.DefaultAuthClientID),
			expected: &operatorv1alpha1.KubermaticAuthConfiguration{},
		},
		{
			yaml:     `kubermatic: { domain: "example.com", auth: { tokenIssuer: "https://example.com/dex" } }`,
			expected: &operatorv1alpha1.KubermaticAuthConfiguration{},
		},
		{
			yaml: `kubermatic: { auth: { clientID: "foo", issuerClientID: "fooIssuer" } }`,
			expected: &operatorv1alpha1.KubermaticAuthConfiguration{
				ClientID: "foo",
			},
		},
		{
			yaml: `kubermatic: { auth: { clientID: "foo", issuerClientID: "bar" } }`,
			expected: &operatorv1alpha1.KubermaticAuthConfiguration{
				ClientID:       "foo",
				IssuerClientID: "bar",
			},
		},
		{
			yaml: `kubermatic: { domain: "example.com", auth: { tokenIssuer: "https://example.com/some/other/url" } }`,
			expected: &operatorv1alpha1.KubermaticAuthConfiguration{
				TokenIssuer: "https://example.com/some/other/url",
			},
		},
		{
			yaml: `kubermatic: { auth: { skipTokenIssuerTLSVerify: "false" } }`,
			expected: &operatorv1alpha1.KubermaticAuthConfiguration{
				SkipTokenIssuerTLSVerify: false,
			},
		},
		{
			yaml: `kubermatic: { auth: { skipTokenIssuerTLSVerify: "nope" } }`,
			expected: &operatorv1alpha1.KubermaticAuthConfiguration{
				SkipTokenIssuerTLSVerify: false,
			},
		},
		{
			yaml: `kubermatic: { auth: { skipTokenIssuerTLSVerify: "true" } }`,
			expected: &operatorv1alpha1.KubermaticAuthConfiguration{
				SkipTokenIssuerTLSVerify: true,
			},
		},
	}

	for idx, testcase := range testcases {
		values := parseYAML(t, testcase.yaml)

		config, err := convertAuth(&values.Kubermatic)
		if err != nil {
			t.Errorf("Test case %d failed, failed to convert: %v", idx+1, err)
			continue
		}

		if !equality.Semantic.DeepEqual(config, testcase.expected) {
			t.Errorf("Test case %d failed:\n%v", idx+1, diff.ObjectDiff(testcase.expected, config))
		}
	}
}

func TestConvertAPI(t *testing.T) {
	testcases := []struct {
		yaml     string
		expected *operatorv1alpha1.KubermaticAPIConfiguration
	}{
		{
			yaml:     `kubermatic: {}`,
			expected: &operatorv1alpha1.KubermaticAPIConfiguration{},
		},
		{
			yaml: `kubermatic: { api: { accessibleAddons: [a, b, c] } }`,
			expected: &operatorv1alpha1.KubermaticAPIConfiguration{
				AccessibleAddons: []string{"a", "b", "c"},
			},
		},
		{
			yaml: `kubermatic: { api: { pprofEndpoint: ":8888" } }`,
			expected: &operatorv1alpha1.KubermaticAPIConfiguration{
				PProfEndpoint: pointer.StringPtr(":8888"),
			},
		},
		{
			yaml:     fmt.Sprintf(`kubermatic: { api: { pprofEndpoint: "%s" } }`, common.DefaultPProfEndpoint),
			expected: &operatorv1alpha1.KubermaticAPIConfiguration{},
		},
	}

	for idx, testcase := range testcases {
		values := parseYAML(t, testcase.yaml)

		config, err := convertAPI(&values.Kubermatic)
		if err != nil {
			t.Errorf("Test case %d failed, failed to convert: %v", idx+1, err)
			continue
		}

		if !equality.Semantic.DeepEqual(config, testcase.expected) {
			t.Errorf("Test case %d failed:\n%v", idx+1, diff.ObjectDiff(testcase.expected, config))
		}
	}
}

func TestConvertPrometheus(t *testing.T) {
	testcases := []struct {
		yaml     string
		expected *operatorv1alpha1.KubermaticUserClusterMonitoringConfiguration
	}{
		{
			yaml:     `kubermatic: {}`,
			expected: &operatorv1alpha1.KubermaticUserClusterMonitoringConfiguration{},
		},
		{
			yaml: `kubermatic: { monitoringScrapeAnnotationPrefix: "foo" }`,
			expected: &operatorv1alpha1.KubermaticUserClusterMonitoringConfiguration{
				ScrapeAnnotationPrefix: "foo",
			},
		},
		{
			yaml: `kubermatic: { clusterNamespacePrometheus: { scrapingConfigs: [a, b, c], rules: { groups: [] } } }`,
			expected: &operatorv1alpha1.KubermaticUserClusterMonitoringConfiguration{
				CustomScrapingConfigs: "- a\n- b\n- c\n",
				CustomRules:           "groups: []\n",
			},
		},
	}

	for idx, testcase := range testcases {
		values := parseYAML(t, testcase.yaml)

		config, err := convertUserCluster(&values.Kubermatic)
		if err != nil {
			t.Errorf("Test case %d failed, failed to convert: %v", idx+1, err)
			continue
		}

		if !equality.Semantic.DeepEqual(&config.Monitoring, testcase.expected) {
			t.Errorf("Test case %d failed:\n%v", idx+1, diff.ObjectDiff(testcase.expected, config.Monitoring))
		}
	}
}

func TestConvertCABundle(t *testing.T) {
	testcases := []struct {
		yaml     string
		expected *operatorv1alpha1.KubermaticConfiguration
	}{
		{
			yaml: `kubermatic: { auth: { caBundle: "aGVsbG8gd29ybGQ=" } }`,
			expected: &operatorv1alpha1.KubermaticConfiguration{
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					CABundle: v1.TypedLocalObjectReference{
						Name: resources.CABundleConfigMapName,
					},
				},
			},
		},
	}

	for idx, testcase := range testcases {
		values := parseYAML(t, testcase.yaml)
		config := &operatorv1alpha1.KubermaticConfiguration{}

		_, err := convertOIDCCABundle(&values, config, "kubermatic")
		if err != nil {
			t.Errorf("Test case %d failed, failed to convert: %v", idx+1, err)
			continue
		}

		if !equality.Semantic.DeepEqual(config, testcase.expected) {
			t.Errorf("Test case %d failed:\n%v", idx+1, diff.ObjectDiff(testcase.expected, config))
		}
	}
}
