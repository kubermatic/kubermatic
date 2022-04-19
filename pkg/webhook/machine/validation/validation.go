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

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"go.uber.org/zap"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/apimachinery/pkg/runtime"
)

// validator for validating Kubermatic Machine CRD.
type validator struct {
	log        *zap.SugaredLogger
	seedClient ctrlruntimeclient.Client
}

// NewValidator returns a new Machine validator.
func NewValidator(seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) *validator {
	log.Named("machine-validator")
	return &validator{
		log:        log,
		seedClient: seedClient,
	}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	machine, ok := obj.(*v1alpha1.Machine)
	if !ok {
		return errors.New("object is not a Machine")
	}

	log := v.log.With("machine", machine.Name)
	log.Debug("validating create")

	return validateQuota(ctx, log, v.seedClient, machine)
}

// ValidateUpdate validates Machine updates. As mutating Machine spec is disallowed by the Machine Mutating webhook,
// no need to check anything here.
func (v *validator) ValidateUpdate(_ context.Context, _, _ runtime.Object) error {
	return nil
}

func (v *validator) ValidateDelete(_ context.Context, _ runtime.Object) error {
	return nil
}
