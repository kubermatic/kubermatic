/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package util

import (
	"context"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// BackupResources takes a GroupVersionKind and dumps all resources
// of that type (across all namespaces) as YAML into the given
// filename (the file will be truncated first).
func BackupResources(ctx context.Context, kubeClient ctrlruntimeclient.Client, gvk schema.GroupVersionKind, filename string) error {
	items, err := ListResources(ctx, kubeClient, gvk)
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	return DumpResources(ctx, filename, items)
}

func ListResources(ctx context.Context, kubeClient ctrlruntimeclient.Client, gvk schema.GroupVersionKind) ([]unstructured.Unstructured, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)

	if err := kubeClient.List(ctx, list); err != nil {
		return nil, err
	}

	return list.Items, nil
}

func DumpResources(ctx context.Context, filename string, objects []unstructured.Unstructured) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", filename, err)
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)

	if _, err := fmt.Fprintf(f, "# This backup was created %s.\n", time.Now().Format(time.UnixDate)); err != nil {
		return fmt.Errorf("failed to write date: %w", err)
	}

	for _, item := range objects {
		if err := encoder.Encode(item.Object); err != nil {
			return fmt.Errorf("failed to encode object as YAML: %w", err)
		}
	}

	return nil
}
