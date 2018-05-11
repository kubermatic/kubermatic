package kubernetes

import (
	"io/ioutil"
	"os"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const registryOverwriteTestTemplate = `{"registry":"{{ overwriteRegistryOrDefault "default" }}"}`

var registryOverwriteTestcases = []struct {
	overwriteRegistry string
	expected          string
}{
	{
		overwriteRegistry: "",
		expected:          `{"registry":"default"}`,
	},
	{
		overwriteRegistry: "overwrite",
		expected:          `{"registry":"overwrite"}`,
	},
}

type FakeV1Obj struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func TestRegistryOverwrite(t *testing.T) {
	templateFileRaw, err := ioutil.TempFile("/tmp", "kubermatic-test")
	if err != nil {
		t.Errorf("Error getting tempfile: %v", err)
	}
	templateFile := templateFileRaw.Name()
	defer func() {
		err = os.Remove(templateFile)
		if err != nil {
			t.Errorf("Error cleaning up tempfile %s after test: %v", templateFile, err)
		}
	}()

	err = ioutil.WriteFile(templateFile, []byte(registryOverwriteTestTemplate), 0600)
	if err != nil {
		t.Errorf("Error writing template to file: %v", err)
	}

	for _, testCase := range registryOverwriteTestcases {
		template, err := ParseFile(templateFile)
		if err != nil {
			t.Errorf("Error executing ParseFile: %v", err)
		}
		SetOverwriteRegistry(testCase.overwriteRegistry)
		fakeV1Obj := &FakeV1Obj{}
		result, err := template.Execute(nil, fakeV1Obj)
		if err != nil {
			t.Errorf("Error executing template: %v", err)
		}
		if testCase.expected != result {
			t.Errorf("Result was '%s' but expected '%s'", result, testCase.expected)
		}
	}
}
