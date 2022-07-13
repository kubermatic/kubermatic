package api

import (
	"context"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8s.io/apimachinery/pkg/util/rand"
	"testing"
)

func TestOidcGroupSupport(t *testing.T) {
	ctx := context.Background()
	t.Log("testing oidc group support")

	masterToken, err := utils.RetrieveMasterToken(ctx)
	if err != nil {
		t.Fatalf("failed to get master token: %v", err)
	}

	testClient := utils.NewTestClient(masterToken, t)
	project, err := testClient.CreateProject(rand.String(10))
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	defer cleanupProject(t, project.ID)
}
