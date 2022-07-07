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
	"errors"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	eegroupprojectbindingvalidation "k8c.io/kubermatic/v2/pkg/ee/validation/groupprojectbinding"

	"k8s.io/apimachinery/pkg/runtime"
)

func validateCreate(_ context.Context,
	_ runtime.Object,
) error {
	return nil
}

func validateUpdate(_ context.Context,
	oldObj runtime.Object,
	newObj runtime.Object,
) error {
	oldGroupProjectBinding, ok := oldObj.(*kubermaticv1.GroupProjectBinding)
	if !ok {
		return errors.New("existing object is not a GroupProjectBinding")
	}

	newGroupProjectBinding, ok := newObj.(*kubermaticv1.GroupProjectBinding)
	if !ok {
		return errors.New("updated object is not a GroupProjectBinding")
	}

	return eegroupprojectbindingvalidation.ValidateUpdate(oldGroupProjectBinding, newGroupProjectBinding)
}

func validateDelete(_ context.Context,
	_ runtime.Object,
) error {
	return nil
}
