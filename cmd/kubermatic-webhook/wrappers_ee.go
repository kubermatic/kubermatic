//go:build ee

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

package main

import (
	"context"

	eewebhook "k8c.io/kubermatic/v3/pkg/ee/cmd/webhook"
	"k8c.io/kubermatic/v3/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func seedGetterFactory(ctx context.Context, client ctrlruntimeclient.Reader, options *appOptions) (provider.SeedGetter, error) {
	return eewebhook.SeedGetterFactory(ctx, client, options.seedName, options.namespace)
}

func seedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (provider.SeedsGetter, error) {
	return eewebhook.SeedsGetterFactory(ctx, client, namespace)
}
