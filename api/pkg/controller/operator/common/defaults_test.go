package common

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/yalp/jsonpath"

	"sigs.k8s.io/yaml"
)

const (
	kubermaticChartPath      = "../../../../../config/kubermatic"
	kubermaticValuesYAMLPath = kubermaticChartPath + "/" + "values.yaml"
)

func getJSONPathValueFromChart(jsonPath string) (interface{}, error) {
	data, err := ioutil.ReadFile(kubermaticValuesYAMLPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %v", kubermaticValuesYAMLPath, err)
	}
	rawData := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("unmarshaling failed: %v", err)
	}
	val, err := jsonpath.Read(rawData, jsonPath)
	if err != nil {
		return nil, fmt.Errorf("read: %v", err)
	}
	return val, nil
}

func getStringDiff(a, b string) []string {
	aLines := strings.Split(strings.TrimSpace(a), "\n")
	bLines := strings.Split(strings.TrimSpace(b), "\n")
	return deep.Equal(aLines, bLines)
}

func TestKubernetesDefaultAddonsMatchesChart(t *testing.T) {
	t.Parallel()
	fromChart, err := getJSONPathValueFromChart("$.kubermatic.controller.addons.kubernetes.defaultAddons")
	if err != nil {
		t.Fatalf("jsonpathValueFromChart: %v", err)
	}
	expected := fmt.Sprintf("%v", kubernetesDefaultAddons)
	actual := fmt.Sprintf("%v", fromChart)
	if actual != expected {
		t.Errorf("kubernetes default addons in the chart (%s) do not match operator defaults (%s)", actual, expected)
	}
}

func TestDefaultAccessibleAddonsMatchChart(t *testing.T) {
	t.Parallel()
	fromChart, err := getJSONPathValueFromChart("$.kubermatic.api.accessibleAddons")
	if err != nil {
		t.Fatalf("jsonpathValueFromChart: %v", err)
	}

	expected := fmt.Sprintf("%v", defaultAccessibleAddons)
	actual := fmt.Sprintf("%v", fromChart)
	if actual != expected {
		t.Errorf("defaultAccessibleAddons from chart (%s) do not match operator defaults (%s)", actual, expected)
	}
}

func TestDefaultBackupStoreContainerMatchesChart(t *testing.T) {
	t.Parallel()
	fromChart, err := ioutil.ReadFile(kubermaticChartPath + "/static/backup-container.yaml")
	if err != nil {
		t.Fatalf("failed to read backupstorecontainer from disk: %v", err)
	}
	if diff := getStringDiff(string(fromChart), defaultBackupStoreContainer); diff != nil {
		t.Errorf("backupcontainer from chart does not match default from operator, diff: %v", diff)
	}
}

func TestDefaultBackupCleanupContainerMatchesChart(t *testing.T) {
	t.Parallel()
	fromChart, err := ioutil.ReadFile(kubermaticChartPath + "/static/cleanup-container.yaml")
	if err != nil {
		t.Fatalf("failed to read backupcleanupcontainer from disk: %v", err)
	}
	if diff := getStringDiff(string(fromChart), defaultBackupCleanupContainer); diff != nil {
		t.Errorf("backupcontainer from chart does not match default from operator, diff: %v", diff)
	}
}

func TestDefaultUIConfigMatchesChart(t *testing.T) {
	t.Parallel()
	fromChart, err := getJSONPathValueFromChart("$.kubermatic.ui.config")
	if err != nil {
		t.Fatalf("jsonpathValueFromChart: %v", err)
	}
	if diff := getStringDiff(fmt.Sprintf("%v", fromChart), defaultUIConfig); diff != nil {
		t.Errorf("defaultUIConfig from chart does not match default from operator, diff: %v", err)
	}
}

func TestDefaultVersionsYAMLMatchesChart(t *testing.T) {
	t.Parallel()
	fromChart, err := ioutil.ReadFile(kubermaticChartPath + "/static/master/versions.yaml")
	if err != nil {
		t.Fatalf("failed to read versions from disk: %v", err)
	}
	if diff := getStringDiff(string(fromChart), defaultVersionsYAML); diff != nil {
		t.Errorf("versions.yaml from chart does not match default from operator, diff: %v", diff)
	}
}

func TestDefaultUpdatesYAMLMatchesChart(t *testing.T) {
	t.Parallel()
	fromChart, err := ioutil.ReadFile(kubermaticChartPath + "/static/master/updates.yaml")
	if err != nil {
		t.Fatalf("failed to read versions from disk: %v", err)
	}
	if diff := getStringDiff(string(fromChart), defaultUpdatesYAML); diff != nil {
		t.Errorf("updates.yaml from chart does not match default from operator, diff: %v", diff)
	}
}

func TestDefaultOpenshiftAddonsMatchesChart(t *testing.T) {
	t.Parallel()
	fromChart, err := ioutil.ReadFile(kubermaticChartPath + "/static/master/openshift-addons.yaml")
	if err != nil {
		t.Fatalf("failed to read versions from disk: %v", err)
	}
	if diff := getStringDiff(string(fromChart), defaultOpenshiftAddons); diff != nil {
		t.Errorf("openshift-addons.yaml from chart does not match default from operator, diff: %v", diff)
	}
}
