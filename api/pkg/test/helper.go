package test

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/pmezard/go-difflib/difflib"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func CompareOutput(t *testing.T, name, output string, update bool, suffix string) {
	filename := name + ".golden"
	if suffix != "" {
		filename += suffix
	}
	golden, err := filepath.Abs(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("failed to get absolute path to goldan file: %v", err)
	}
	if update {
		if err := ioutil.WriteFile(golden, []byte(output), 0644); err != nil {
			t.Fatalf("failed to write updated fixture: %v", err)
		}
	}
	expected, err := ioutil.ReadFile(golden)
	if err != nil {
		t.Fatalf("failed to read .golden file: %v", err)
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(expected)),
		B:        difflib.SplitLines(output),
		FromFile: "Fixture",
		ToFile:   "Current",
		Context:  3,
	}
	diffStr, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		t.Fatal(err)
	}

	if diffStr != "" {
		t.Errorf("got diff between expected and actual result: \n%s\n", diffStr)
	}
}

type CredentialsData struct {
	KubermaticCluster *kubermaticv1.Cluster
	Client            ctrlruntimeclient.Client
}

// Cluster returns the cluster
func (d CredentialsData) Cluster() *kubermaticv1.Cluster {
	return d.KubermaticCluster
}

func (d CredentialsData) GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	// We need all three of these to fetch and use a secret
	if configVar.Name != "" && configVar.Namespace != "" && key != "" {
		secret := &corev1.Secret{}
		namespacedName := types.NamespacedName{Namespace: configVar.Namespace, Name: configVar.Name}
		if err := d.Client.Get(context.Background(), namespacedName, secret); err != nil {
			return "", fmt.Errorf("error retrieving secret %q from namespace %q: %v", configVar.Name, configVar.Namespace, err)
		}

		if val, ok := secret.Data[key]; ok {
			return string(val), nil
		}
		return "", fmt.Errorf("secret %q in namespace %q has no key %q", configVar.Name, configVar.Namespace, key)
	}
	return "", nil
}
