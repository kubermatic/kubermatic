package kubernetes_test

import (
	"context"
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestListApplicationDefinitions(t *testing.T) {
	appDef1 := test.GenApplicationDefinition("app1")
	appDef2 := test.GenApplicationDefinition("app2")

	testcases := []struct {
		name            string
		existingAppDefs []*appskubermaticv1.ApplicationDefinition
		expectedAppDefs []*appskubermaticv1.ApplicationDefinition
	}{
		{
			name:            "list all applicationdefinitions",
			existingAppDefs: []*appskubermaticv1.ApplicationDefinition{appDef1, appDef2},
			expectedAppDefs: []*appskubermaticv1.ApplicationDefinition{appDef1, appDef2},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			kubermaticObjects := []ctrlruntimeclient.Object{}
			for _, binding := range tc.existingAppDefs {
				kubermaticObjects = append(kubermaticObjects, binding)
			}
			fakeClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(kubermaticObjects...).
				Build()

			// act
			target := kubernetes.NewApplicationDefinitionProvider(fakeClient)
			result, err := target.ListUnsecured(context.Background())

			// validate
			if err != nil {
				t.Fatal(err)
			}
			if len(tc.expectedAppDefs) != len(result.Items) {
				t.Fatalf("expected to get %d appdefs, but got %d", len(tc.expectedAppDefs), len(result.Items))
			}
		})
	}
}
