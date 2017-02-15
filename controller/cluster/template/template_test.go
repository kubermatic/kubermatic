package template

import (
	"testing"
	texttemplate "text/template"
)

type testData struct {
	Name string `yaml:"name"`
}

func TestParseFiles(t *testing.T) {
	path := "fixtures/simple.yml"

	if _, err := ParseFiles(path); err != nil {
		t.Error(err)
	}
}

func TestTemplate_Execute(t *testing.T) {
	path := "fixtures/simple.yml"
	txt, err := texttemplate.ParseFiles(path)
	if err != nil {
		t.Fatal(err)
	}

	tpl := &Template{txt}
	source := &testData{Name: "Günther"}

	result := &testData{}
	if err := tpl.Execute(source, result); err != nil {
		t.Error(err)
		return
	}

	if result.Name != "Günther" {
		t.Errorf("Expected rednered template to have name Günther, got %s", result.Name)
	}
}

func TestTemplate_ExecuteFailsWithInvalidContent(t *testing.T) {
	path := "fixtures/invalid.yml"
	txt, err := texttemplate.ParseFiles(path)
	if err != nil {
		t.Fatal(err)
	}

	tpl := &Template{txt}
	source := &testData{Name: "Günther"}
	result := &testData{}

	if err := tpl.Execute(source, result); err == nil {
		t.Error("Execute finished without returning an error on an invalid file")
	} else if err.Error() != "failed converting yaml to json: yaml: line 1: mapping values are not allowed in this context" {
		t.Errorf("Expected to get an yaml mapping error, instead got: %s", err.Error())
	}
}

func TestTemplate_ExecuteFailsWithNotExistingFile(t *testing.T) {
	path := "fixtures/not-existing.yml"
	_, err := texttemplate.ParseFiles(path)
	if err == nil {
		t.Fatal("Expected to get an error, got nil")
	}

	if err.Error() != "open fixtures/not-existing.yml: no such file or directory" {
		t.Fatalf("Expected to get no such file or directory, got %q", err.Error())
	}
}
