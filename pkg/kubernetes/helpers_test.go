package kubernetes

import (
	"testing"
)

func TestGenerateToken(t *testing.T) {
	tokenA := GenerateToken()
	tokenB := GenerateToken()

	if len(tokenA) == 0 {
		t.Error("generated token is empty")
	}

	if tokenA == tokenB {
		t.Errorf("two new tokens should not be identical, but are: '%s'", tokenA)
	}

	if err := ValidateKubernetesToken(tokenA); err != nil {
		t.Errorf("generated token is malformed: %v", err)
	}
}
