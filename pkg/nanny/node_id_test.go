package nanny

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestLoadNodeID(t *testing.T) {
	UID := "8f40cf6b-55f3-48bd-83f6-e721aa864afe"
	f, err := ioutil.TempFile("", "node_id")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err := os.Remove(f.Name())
		if err != nil {
			t.Fatalf("Failed to remove temp file: %v", err)
		}
	}()

	if _, err := f.Write([]byte(UID)); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	nodeID, err := LoadNodeID(f.Name())
	if err != nil {
		t.Fatalf("Expected method to not throw an error, got %v", err)
	}

	if nodeID != UID {
		t.Errorf("Expected nodeID to be %q, got %q", UID, nodeID)
	}
}
