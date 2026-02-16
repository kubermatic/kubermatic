//go:build ee

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

package validation

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	eeresourcequotavalidation "k8c.io/kubermatic/v2/pkg/ee/validation/resourcequota"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func validateCreate(ctx context.Context,
	obj *kubermaticv1.ResourceQuota,
	client ctrlruntimeclient.Client,
) error {
	return eeresourcequotavalidation.ValidateCreate(ctx, obj, client)
}

func validateUpdate(ctx context.Context,
	oldObj *kubermaticv1.ResourceQuota,
	newObj *kubermaticv1.ResourceQuota,
) error {
	return eeresourcequotavalidation.ValidateUpdate(ctx, oldObj, newObj)
}

func validateDelete(ctx context.Context,
	obj *kubermaticv1.ResourceQuota,
	client ctrlruntimeclient.Client,
) error {
	return eeresourcequotavalidation.ValidateDelete(ctx, obj, client)
}
