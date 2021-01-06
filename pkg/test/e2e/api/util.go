package api

import (
	"context"
	"testing"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
)

func cleanupProject(t *testing.T, id string) {
	// use a dedicated context so that cleanups always run, even
	// if the context inside a test was already cancelled
	token, err := utils.RetrieveAdminMasterToken(context.Background())
	if err != nil {
		t.Fatalf("failed to get master token: %v", err)
	}

	utils.NewTestClient(token, t).CleanupProject(t, id)
}
