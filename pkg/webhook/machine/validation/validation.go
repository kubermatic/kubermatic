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

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Kubermatic Machine CRD.
type validator struct {
	log        *zap.SugaredLogger
	seedClient ctrlruntimeclient.Client
	userClient ctrlruntimeclient.Client
	caBundle   *certificates.CABundle
}

// NewValidator returns a new Machine validator.
func NewValidator(seedClient, userClient ctrlruntimeclient.Client, log *zap.SugaredLogger, caBundle *certificates.CABundle) *validator {
	return &validator{
		log:        log,
		seedClient: seedClient,
		userClient: userClient,
		caBundle:   caBundle,
	}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	machine, ok := obj.(*clusterv1alpha1.Machine)
	if !ok {
		return errors.New("object is not a Machine")
	}

	log := v.log.With("machine", machine.Name)
	log.Debug("validating create")

	quota := getResourceQuota()
	if quota != nil {
		return validateQuota(ctx, log, v.seedClient, v.userClient, machine, v.caBundle)
	}
	return nil
}

// ValidateUpdate validates Machine updates. As mutating Machine spec is disallowed by the Machine Mutating webhook,
// no need to check anything here.
func (v *validator) ValidateUpdate(_ context.Context, _, _ runtime.Object) error {
	return nil
}

func (v *validator) ValidateDelete(_ context.Context, _ runtime.Object) error {
	return nil
}

// Gets resource quota for the project. Not implemented yet because the ResourceQuota CRD is not implemented.
// For now this just stops resource quota check, as there are no resource quotas.
// TODO implement when ResourceQuota CRD is available.
func getResourceQuota() runtime.Object {
	return nil
}
