//go:build e2e

/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package argocd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	yamlutil "k8c.io/kubermatic/v2/pkg/util/yaml"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	userconfig  string
	credentials jig.AWSCredentials
	logOptions  = utils.DefaultLogOptions
)

const (
	projectName = "cilium-test-project"
	argoCDNs    = "argocd"
	argoCDName  = "argocd"
)

func init() {
	flag.StringVar(&userconfig, "userconfig", "", "path to kubeconfig of usercluster")
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestInExistingCluster(t *testing.T) {
	if userconfig == "" {
		t.Logf("kubeconfig for usercluster not provided, test passes vacuously.")
		t.Logf("to run against an existing usercluster use following command:")
		t.Logf("go test ./pkg/test/e2e/default-app-catalog-applications-test/argocd -v -tags e2e -timeout 30m -run TestInExistingCluster -userconfig <USERCLUSTER KUBECONFIG>")
		return
	}

	rawLog := log.NewFromOptions(logOptions)
	logger := rawLog.Sugar()

	config, err := clientcmd.BuildConfigFromFlags("", userconfig)
	if err != nil {
		t.Fatalf("failed to build config: %v", err)
	}

	client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		t.Fatalf("failed to build ctrlruntime client: %v", err)
	}

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	testUserCluster(context.Background(), t, logger, client)
}

func TestArgoCDClusters(t *testing.T) {
	rawLog := log.NewFromOptions(logOptions)
	logger := rawLog.Sugar()
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

	//testAppDefs, err := resourcesFromYaml("./testdata/argocd-app.yaml")
	//if err != nil {
	//	t.Fatalf("failed to read objects from yaml: %v", err)
	//}
	//for _, testAppDef := range testAppDefs {
	//	if err := seedClient.Create(ctx, testAppDef); err != nil {
	//		t.Fatalf("failed to apply resource: %v", err)
	//	}
	//
	//	logger.Infow("Created object", "kind", testAppDef.GetObjectKind(), "name", testAppDef.GetName())
	//}

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	client, cleanup, tLogger, err := createUserCluster(ctx, t, logger, seedClient)
	if cleanup != nil {
		defer cleanup()
	}

	if err != nil {
		t.Fatalf("failed to create user cluster: %v", err)
	}

	installArgoCDTests(ctx, t, tLogger, seedClient)

	testUserCluster(ctx, t, tLogger, client)

}

//gocyclo:ignore
func testUserCluster(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client) {
	log.Info("Running ArgoCD tests...")
	//ns := corev1.Namespace{}
	//ns.Name = argoCDNs
	//err := client.Create(ctx, &ns)
	//if err != nil {
	//	//t.Fatalf("failed to create %q namespace: %v", argoCDNs, err)
	//}
	//defer func() {
	//	err := client.Delete(ctx, &ns)
	//	if err != nil {
	//		t.Fatalf("failed to delete %q namespace: %v", argoCDNs, err)
	//	}
	//}()
	//
	//log = log.With("namespace", argoCDNs)
	//log.Debug("Namespace created")

	//installArgoCDTests(ctx, t, log, client)

	log.Info("Waiting for ArgoCD pods to get ready...")
	err := waitForPods(ctx, t, log, client, argoCDNs, "name", []string{
		"argocd",
	})
	if err != nil {
		t.Fatalf("pods never became ready: %v", err)
	}

	log.Info("Checking for argocd ApplicationInstallation...")
	err = wait.PollLog(ctx, log, 2*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		app := &appskubermaticv1.ApplicationInstallation{}
		if err := client.Get(context.Background(), types.NamespacedName{Namespace: "argocd", Name: "argocd"}, app); err != nil {
			return fmt.Errorf("failed to get ArgoCD ApplicationInstallation in user cluster: %w", err), nil
		}
		if app.Status.ApplicationVersion == nil {
			return fmt.Errorf("application not yet installed: %v", app.Status), nil
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("Application observe test failed: %v", err)
	}
}

func waitForPods(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client, namespace string, key string, names []string) error {
	log = log.With("namespace", namespace)

	r, err := labels.NewRequirement(key, selection.In, names)
	if err != nil {
		return fmt.Errorf("failed to build requirement: %w", err)
	}
	l := labels.NewSelector().Add(*r)

	return wait.PollLog(ctx, log, 5*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		pods := corev1.PodList{}
		err = client.List(ctx, &pods, ctrlruntimeclient.InNamespace(namespace), ctrlruntimeclient.MatchingLabelsSelector{Selector: l})
		if err != nil {
			return fmt.Errorf("failed to list Pods: %w", err), nil
		}

		if len(pods.Items) == 0 {
			return errors.New("no Pods found"), nil
		}

		unready := sets.New[string]()
		for _, pod := range pods.Items {
			ready := false
			for _, c := range pod.Status.Conditions {
				if c.Type == corev1.ContainersReady {
					ready = c.Status == corev1.ConditionTrue
				}
			}

			if !ready {
				unready.Insert(pod.Name)
			}
		}

		if unready.Len() > 0 {
			return fmt.Errorf("not all Pods are ready: %v", sets.List(unready)), nil
		}

		return nil, nil
	})
}

func installArgoCDTests(ctx context.Context, t *testing.T, log *zap.SugaredLogger, client ctrlruntimeclient.Client) {
	ns := corev1.Namespace{}
	ns.Name = argoCDNs
	err := client.Create(ctx, &ns)
	if err != nil {
		t.Fatalf("failed to create %q namespace: %v", argoCDNs, err)
	}
	defer func() {
		err := client.Delete(ctx, &ns)
		if err != nil {
			t.Fatalf("failed to delete %q namespace: %v", argoCDNs, err)
		}
	}()

	objs, err := resourcesFromYaml("./testdata/argocd-app.yaml")
	if err != nil {
		t.Fatalf("failed to read objects from yaml: %v", err)
	}

	for _, obj := range objs {
		obj.SetNamespace(argoCDNs)
		if err := client.Create(ctx, obj); err != nil {
			t.Fatalf("failed to apply resource: %v", err)
		}

		log.Info("installed resources")
		log.Debugw("Created object", "kind", obj.GetObjectKind(), "name", obj.GetName())
	}
}

func resourcesFromYaml(filename string) ([]ctrlruntimeclient.Object, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	manifests, err := yamlutil.ParseMultipleDocuments(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var objs []ctrlruntimeclient.Object
	for _, m := range manifests {
		obj := &unstructured.Unstructured{}
		if err := kyaml.NewYAMLOrJSONDecoder(bytes.NewReader(m.Raw), 1024).Decode(obj); err != nil {
			return nil, err
		}

		objs = append(objs, obj)
	}

	return objs, nil
}

func getTestApplicationAnnotation(appName string) ([]byte, error) {
	var values json.RawMessage
	err := json.Unmarshal([]byte(`{"controller":{"ingressClass":"test-nginx"}}`), &values)
	if err != nil {
		return nil, err
	}

	app := apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name: appName,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name: metav1.NamespaceSystem,
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    appName,
				Version: "1.8.1",
			},
			Values: values,
		},
	}
	applications := []apiv1.Application{app}
	data, err := json.Marshal(applications)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// creates a usercluster on aws.
func createUserCluster(
	ctx context.Context,
	t *testing.T,
	log *zap.SugaredLogger,
	masterClient ctrlruntimeclient.Client,
) (ctrlruntimeclient.Client, func(), *zap.SugaredLogger, error) {
	testAppAnnotation, err := getTestApplicationAnnotation(argoCDName)
	if err != nil {
		return nil, nil, log, fmt.Errorf("failed to prepare test application: %w", err)
	}

	testJig := jig.NewAWSCluster(masterClient, log, credentials, 2, nil)
	testJig.ProjectJig.WithHumanReadableName(projectName)
	testJig.ClusterJig.
		WithTestName("argocd").
		WithKonnectivity(true).
		WithAnnotations(map[string]string{
			kubermaticv1.InitialApplicationInstallationsRequestAnnotation: string(testAppAnnotation),
		})

	cleanup := func() {
		testJig.Cleanup(ctx, t, true)
	}

	// let the magic happen
	if _, _, err := testJig.Setup(ctx, jig.WaitForReadyPods); err != nil {
		return nil, cleanup, log, fmt.Errorf("failed to setup test environment: %w", err)
	}

	clusterClient, err := testJig.ClusterClient(ctx)

	ns := corev1.Namespace{}
	ns.Name = argoCDNs
	err = clusterClient.Create(ctx, &ns)
	if err != nil {
		t.Fatalf("failed to create %q namespace: %v", argoCDNs, err)
	}
	defer func() {
		err := clusterClient.Delete(ctx, &ns)
		if err != nil {
			t.Fatalf("failed to delete %q namespace: %v", argoCDNs, err)
		}
	}()

	log = log.With("namespace", argoCDNs)
	log.Debug("Namespace created")

	return clusterClient, cleanup, log, err
}
