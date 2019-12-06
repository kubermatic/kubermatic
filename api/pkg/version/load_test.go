package version

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestLoadUpdatesAutomaticNodeUpdateSetsUpdateToTrue(t *testing.T) {
	fileContent := []byte(`
updates:
- from: 1.12.*
  to: 1.13.*
  automaticNodeUpdate: true
`)
	file, err := ioutil.TempFile("/tmp", "kubermatic-test")
	if err != nil {
		t.Fatalf("failed to create tempfile: %v", err)
	}
	defer file.Close()
	defer os.Remove(file.Name())

	if _, err := file.Write(fileContent); err != nil {
		t.Fatalf("failed to write to tempfile: %v", err)
	}

	updates, err := LoadUpdates(file.Name())
	if err != nil {
		t.Fatalf("failed to load updates file: %v", err)
	}
	if n := len(updates); n != 1 {
		t.Fatalf("expected to get exactly one update, got %d", n)
	}
	if !updates[0].Automatic {
		t.Fatal("Setting automaticNodeUpdate: true didn't result in automatic: true")
	}
}
