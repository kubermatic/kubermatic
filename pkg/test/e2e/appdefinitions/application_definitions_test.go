//go:build e2e

/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package defaultappcatalog

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	credentials jig.AWSCredentials
	logOptions  = utils.DefaultLogOptions
)

const (
	projectName                    = "app-definitions-test-project"
	countOfInstalledAppDefinitions = 5
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestClusters(t *testing.T) {
	rawLog := log.NewFromOptions(logOptions)
	ctx := context.Background()

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		t.Fatalf("failed to build config: %v", err)
	}

	seedClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		t.Fatalf("failed to build ctrlruntime client: %v", err)
	}

	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	appDefsList := &appskubermaticv1.ApplicationDefinitionList{}
	listOptions := &ctrlruntimeclient.ListOptions{
		Namespace: "", // all namespaces
	}
	if err := seedClient.List(ctx, appDefsList, listOptions); err != nil {
		t.Fatalf("failed to get ApplicationDefinitions in seed cluster: %v", err)
	}

	fmt.Printf("Found %d ApplicationDefinitions:\n", len(appDefsList.Items))
	for _, appDef := range appDefsList.Items {
		t.Logf("- Name: %s, Namespace: %s\n", appDef.Name, appDef.Namespace)
	}

	if len(appDefsList.Items) != countOfInstalledAppDefinitions {
		t.Fatalf("the number of the applications definitions in the seed cluster is not correct. Expected: %d, Got: %d", countOfInstalledAppDefinitions, len(appDefsList.Items))
	}
}
