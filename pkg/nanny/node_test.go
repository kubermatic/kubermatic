package nanny

import (
	"testing"
)

func TestNewNodeFromSystemData(t *testing.T) {
	UID := "test"

	n, err := NewNodeFromSystemData(UID)
	if err != nil {
		t.Fatal(err)
	}

	if n.Space == 0 {
		t.Errorf("Expected Space to be greater then 0, got %d", n.Space)
	}

	if n.Memory == 0 {
		t.Errorf("Expected Memory to be greater then 0, got %d", n.Memory)
	}

	if len(n.CPUs) == 0 {
		t.Errorf("Expected number of CPUs to be greater then 0, got %d", len(n.CPUs))
	}

	for i, c := range n.CPUs {
		if c.Cores == 0 {
			t.Errorf("Expected number of cores of CPU %d to be greater then 0, got %d", i, c.Cores)
		}

		if c.Frequency <= 0 {
			t.Errorf("Expected frequency of CPU %d to be greater then 0, got %f", i, c.Frequency)
		}
	}
}
