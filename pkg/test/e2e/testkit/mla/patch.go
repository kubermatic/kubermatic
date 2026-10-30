/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package mla

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"

	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// patchWithRevert applies mutate to the current state of obj as a JSON merge
// patch that only holds the changed fields, and returns a function that
// applies the inverse patch. Fields the caller did not touch, including ones a
// controller changes in between, are left alone on revert.
func patchWithRevert(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object, mutate func()) (func(context.Context) error, error) {
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(obj), obj); err != nil {
		return nil, fmt.Errorf("failed to get %T %s: %w", obj, obj.GetName(), err)
	}

	original, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	mutate()

	modified, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	forward, err := jsonpatch.CreateMergePatch(original, modified)
	if err != nil {
		return nil, fmt.Errorf("failed to compute patch: %w", err)
	}

	reverse, err := jsonpatch.CreateMergePatch(modified, original)
	if err != nil {
		return nil, fmt.Errorf("failed to compute revert patch: %w", err)
	}

	if err := client.Patch(ctx, obj, ctrlruntimeclient.RawPatch(types.MergePatchType, forward)); err != nil {
		return nil, fmt.Errorf("failed to patch %T %s: %w", obj, obj.GetName(), err)
	}

	target := obj.DeepCopyObject().(ctrlruntimeclient.Object)

	return func(ctx context.Context) error {
		if err := client.Patch(ctx, target, ctrlruntimeclient.RawPatch(types.MergePatchType, reverse)); err != nil {
			return fmt.Errorf("failed to revert %T %s: %w", target, target.GetName(), err)
		}

		return nil
	}, nil
}

// revertContext is used for reverts, which must also run when the check's own
// context was cancelled or timed out.
func revertContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Minute)
}

// cortexQuery runs an instant PromQL query against Cortex for the cluster's
// tenant, bypassing the MLA gateway.
func cortexQuery(ctx context.Context, in Input, query string) ([]promVectorSample, error) {
	return cortexQueryWithParams(ctx, in, query, nil)
}

func cortexQueryWithParams(ctx context.Context, in Input, query string, params url.Values) ([]promVectorSample, error) {
	params = cloneValues(params)
	params.Set("query", query)

	proxy, err := newServiceProxy(in.SeedConfig, mlaNamespace, cortexQueryFrontendService, cortexQueryFrontendPort)
	if err != nil {
		return nil, err
	}

	header := http.Header{tenantHeader: []string{in.Cluster.Name}}

	code, body, err := proxy.do(ctx, http.MethodGet, "/prometheus/api/v1/query?"+params.Encode(), header, nil)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("query %s: expected HTTP 200, got HTTP %d: %s", query, code, body)
	}

	response := promAPIResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unable to decode response: %w", err)
	}
	if response.Status != "success" {
		return nil, fmt.Errorf("query %s failed: %s", query, response.Error)
	}

	vector := promVector{}
	if err := json.Unmarshal(response.Data, &vector); err != nil {
		return nil, fmt.Errorf("unable to decode result: %w", err)
	}

	return vector.Result, nil
}

func cloneValues(values url.Values) url.Values {
	clone := url.Values{}
	for key, value := range values {
		clone[key] = append([]string(nil), value...)
	}

	return clone
}
