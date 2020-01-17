package conversion

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"

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
			yaml: `kubermatic: { auth: { caBundle: "aGVsbG8gd29ybGQ=" } }`,
			expected: &operatorv1alpha1.KubermaticAuthConfiguration{
				CABundle: "hello world",
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
