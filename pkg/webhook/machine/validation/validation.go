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
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Kubermatic Machine CRD.
type validator struct {
	log             *zap.SugaredLogger
	seedClient      ctrlruntimeclient.Client
	userClient      ctrlruntimeclient.Client
	caBundle        *certificates.CABundle
	subjectSelector labels.Selector
}

// NewValidator returns a new Machine validator.
func NewValidator(seedClient, userClient ctrlruntimeclient.Client, log *zap.SugaredLogger, caBundle *certificates.CABundle,
	projectID string) (*validator, error) {
	subjectNameReq, err := labels.NewRequirement(kubermaticv1.ResourceQuotaSubjectNameLabelKey, selection.Equals, []string{projectID})
	if err != nil {
		return nil, fmt.Errorf("error creating resource quota subject name requirement: %w", err)
	}
	subjectKindReq, err := labels.NewRequirement(kubermaticv1.ResourceQuotaSubjectKindLabelKey, selection.Equals, []string{kubermaticv1.ProjectSubjectKind})
	if err != nil {
		return nil, fmt.Errorf("error creating resource quota subject kind requirement: %w", err)
	}
	subjectSelector := labels.NewSelector().Add(*subjectNameReq, *subjectKindReq)

	return &validator{
		log:             log,
		seedClient:      seedClient,
		userClient:      userClient,
		caBundle:        caBundle,
		subjectSelector: subjectSelector,
	}, nil
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	machine, ok := obj.(*clusterv1alpha1.Machine)
	if !ok {
		return nil, errors.New("object is not a Machine")
	}

	log := v.log.With("machine", machine.Name)
	log.Debug("validating create")

	quota, err := getResourceQuota(ctx, v.seedClient, v.subjectSelector)
	if err != nil {
		return nil, err
	}
	if quota != nil {
		return nil, validateQuota(ctx, log, v.userClient, machine, v.caBundle, quota)
	}
	return nil, nil
}

// ValidateUpdate validates Machine updates. As mutating Machine spec is disallowed by the Machine Mutating webhook,
// no need to check anything here.
func (v *validator) ValidateUpdate(_ context.Context, _, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *validator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
