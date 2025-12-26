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

package validation

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
	"k8c.io/kubermatic/v2/pkg/util/kyverno"
	"k8c.io/kubermatic/v2/pkg/validation"
	"k8c.io/kubermatic/v2/pkg/version"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator for validating Kubermatic Cluster CRD.
type validator struct {
	features     features.FeatureGate
	client       ctrlruntimeclient.Client
	seedGetter   provider.SeedGetter
	configGetter provider.KubermaticConfigurationGetter
	caBundle     *x509.CertPool

	// disableProviderValidation is only for unit tests, to ensure no
	// provider would phone home to validate dummy test credentials
	disableProviderValidation bool
}

// NewValidator returns a new cluster validator.
func NewValidator(client ctrlruntimeclient.Client, seedGetter provider.SeedGetter, configGetter provider.KubermaticConfigurationGetter, features features.FeatureGate, caBundle *x509.CertPool) *validator {
	return &validator{
		client:       client,
		features:     features,
		seedGetter:   seedGetter,
		configGetter: configGetter,
		caBundle:     caBundle,
	}
}

var _ admission.CustomValidator = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cluster, ok := obj.(*kubermaticv1.Cluster)
	if !ok {
		return nil, errors.New("object is not a Cluster")
	}

	// This validates the charset and the max length.
	if errs := k8svalidation.IsDNS1035Label(cluster.Name); len(errs) != 0 {
		return nil, fmt.Errorf("cluster name must be valid rfc1035 label: %s", strings.Join(errs, ","))
	}

	if len(cluster.Name) > validation.MaxClusterNameLength {
		return nil, fmt.Errorf("cluster name exceeds maximum allowed length of %d characters", validation.MaxClusterNameLength)
	}

	datacenter, seed, cloudProvider, err := v.buildValidationDependencies(ctx, cluster)
	if err != nil {
		return nil, err
	}

	config, configErr := v.configGetter(ctx)
	if configErr != nil {
		return nil, configErr
	}

	versionManager := version.NewFromConfiguration(config)

	errs := validation.ValidateNewClusterSpec(ctx, &cluster.Spec, datacenter, seed, cloudProvider, versionManager, v.features, nil)

	if err := v.validateProjectRelation(ctx, cluster, nil); err != nil {
		errs = append(errs, err)
	}

	if err := v.validateKyvernoEnforcement(cluster, nil, datacenter, seed, config); err != nil {
		errs = append(errs, err)
	}

	return nil, errs.ToAggregate()
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	oldCluster, ok := oldObj.(*kubermaticv1.Cluster)
	if !ok {
		return nil, errors.New("old object is not a Cluster")
	}

	newCluster, ok := newObj.(*kubermaticv1.Cluster)
	if !ok {
		return nil, errors.New("new object is not a Cluster")
	}

	datacenter, seed, cloudProvider, err := v.buildValidationDependencies(ctx, newCluster)
	if err != nil {
		return nil, err
	}

	config, configErr := v.configGetter(ctx)
	if configErr != nil {
		return nil, configErr
	}

	updateManager := version.NewFromConfiguration(config)

	errs := validation.ValidateClusterUpdate(ctx, newCluster, oldCluster, datacenter, seed, cloudProvider, updateManager, v.features)

	if err := v.validateProjectRelation(ctx, newCluster, oldCluster); err != nil {
		errs = append(errs, err)
	}

	if err := v.validateKyvernoEnforcement(newCluster, oldCluster, datacenter, seed, config); err != nil {
		errs = append(errs, err)
	}

	return nil, errs.ToAggregate()
}

func (v *validator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *validator) buildValidationDependencies(ctx context.Context, c *kubermaticv1.Cluster) (*kubermaticv1.Datacenter, *kubermaticv1.Seed, provider.CloudProvider, *field.Error) {
	seed, err := v.seedGetter()
	if err != nil {
		return nil, nil, nil, field.InternalError(nil, err)
	}
	if seed == nil {
		return nil, nil, nil, field.InternalError(nil, errors.New("webhook is not configured with -seed-name, cannot validate Clusters"))
	}

	datacenter, fieldErr := defaulting.DatacenterForClusterSpec(&c.Spec, seed)
	if fieldErr != nil {
		return nil, nil, nil, fieldErr
	}

	if v.disableProviderValidation {
		return datacenter, seed, nil, nil
	}

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, v.client)
	cloudProvider, err := cloud.Provider(datacenter, secretKeySelectorFunc, v.caBundle)
	if err != nil {
		return nil, nil, nil, field.InternalError(nil, err)
	}

	return datacenter, seed, cloudProvider, nil
}

func (v *validator) validateProjectRelation(ctx context.Context, cluster *kubermaticv1.Cluster, oldCluster *kubermaticv1.Cluster) *field.Error {
	label := kubermaticv1.ProjectIDLabelKey
	fieldPath := field.NewPath("metadata", "labels")
	isUpdate := oldCluster != nil

	if isUpdate && cluster.Labels[label] != oldCluster.Labels[label] {
		return field.Invalid(fieldPath, cluster.Labels[label], fmt.Sprintf("the %s label is immutable", label))
	}

	projectID := cluster.Labels[label]
	if projectID == "" {
		return field.Required(fieldPath, fmt.Sprintf("Cluster resources must have a %q label", label))
	}

	project := &kubermaticv1.Project{}
	if err := v.client.Get(ctx, types.NamespacedName{Name: projectID}, project); err != nil {
		if apierrors.IsNotFound(err) {
			// during cluster creation, we enforce the project label;
			// during updates we are more relaxed and only require that the label isn't changed,
			// so that if a project gets removed before the cluster (for whatever reason), then
			// the cluster cleanup can still progress and is not blocked by the webhook rejecting
			// the stale label
			if isUpdate {
				return nil
			}

			return field.Invalid(fieldPath, projectID, "no such project exists")
		}

		return field.InternalError(fieldPath, fmt.Errorf("failed to get project: %w", err))
	}

	// Do not check the project phase, as projects only get Active after being successfully
	// reconciled. This requires the owner user to be setup properly as well, which in turn
	// requires owner references to be setup. All of this is super annoying when doing
	// GitOps. Instead we rely on _eventual_ consistency and only check that the project
	// exists and is not being deleted.
	if !isUpdate && project.DeletionTimestamp != nil {
		return field.Invalid(fieldPath, projectID, "project is in deletion, cannot create new clusters in it")
	}

	return nil
}

// validateKyvernoEnforcement ensures users cannot override enforced Kyverno settings through Cluster spec.
func (v *validator) validateKyvernoEnforcement(
	newCluster, oldCluster *kubermaticv1.Cluster,
	datacenter *kubermaticv1.Datacenter,
	seed *kubermaticv1.Seed,
	config *kubermaticv1.KubermaticConfiguration,
) *field.Error {
	enforcementInfo := kyverno.GetEnforcement(
		datacenter.Spec.Kyverno,
		seed.Spec.Kyverno,
		config.Spec.UserCluster.Kyverno,
	)
	if !enforcementInfo.Enforced {
		return nil
	}

	fieldPath := field.NewPath("spec", "kyverno", "enabled")

	isUpdate := oldCluster != nil
	if isUpdate {
		if oldCluster.Spec.IsKyvernoEnabled() && !newCluster.Spec.IsKyvernoEnabled() {
			return field.Invalid(
				fieldPath,
				newCluster.Spec.Kyverno,
				fmt.Sprintf("kyverno is enforced by %q and cannot be disabled", enforcementInfo.Source),
			)
		}

		return nil
	}

	if !newCluster.Spec.IsKyvernoEnabled() {
		return field.Invalid(
			fieldPath,
			newCluster.Spec.Kyverno,
			fmt.Sprintf("kyverno is enforced by %q and must be enabled for new clusters", enforcementInfo.Source),
		)
	}

	return nil
}
