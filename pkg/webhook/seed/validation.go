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

package seed

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sync"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/validation"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type validator struct {
	seedsGetter      provider.SeedsGetter
	seedClientGetter provider.SeedClientGetter
	features         features.FeatureGate
	lock             *sync.Mutex
}

func newSeedValidator(
	seedsGetter provider.SeedsGetter,
	seedClientGetter provider.SeedClientGetter,
	features features.FeatureGate,
) (*validator, error) {
	return &validator{
		seedsGetter:      seedsGetter,
		seedClientGetter: seedClientGetter,
		features:         features,
		lock:             &sync.Mutex{},
	}, nil
}

var resourceNameValidator = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)

var _ admission.Validator[*kubermaticv1.Seed] = &validator{}

func (v *validator) ValidateCreate(ctx context.Context, obj *kubermaticv1.Seed) (admission.Warnings, error) {
	return nil, v.validate(ctx, obj, false)
}

func (v *validator) ValidateUpdate(ctx context.Context, oldObj, newObj *kubermaticv1.Seed) (admission.Warnings, error) {
	return nil, v.validate(ctx, newObj, false)
}

func (v *validator) ValidateDelete(ctx context.Context, obj *kubermaticv1.Seed) (admission.Warnings, error) {
	return nil, v.validate(ctx, obj, true)
}

func (v *validator) validate(ctx context.Context, subject *kubermaticv1.Seed, isDelete bool) error {
	// We need locking to make the validation concurrency-safe
	// TODO: this is acceptable as request rate is low, but is it required?
	v.lock.Lock()
	defer v.lock.Unlock()

	// fetch all relevant Seeds; in CE this will only be the one supported Seed,
	// in EE this is a map of all Seed resources; since in CE naming a Seed
	// anything but "kubermatic" is forbidden, it's okay that we use the default
	// SeedsGetter, which will also only return this one Seed (i.e. the webhook
	// in CE never sees any other Seeds)
	existingSeeds, err := v.seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to list Seeds: %w", err)
	}

	if isDelete {
		// when a namespace is deleted, a DELETE call for all Seeds in the namespace
		// is issued; this request has no .Request.Name set, so this check will make
		// sure that we exit cleanly and allow deleting namespaces without seeds
		if _, exists := existingSeeds[subject.Name]; !exists {
			return nil
		}

		// in case of delete request the seed is empty, so fetch the current one from
		// the cluster instead
		subject = existingSeeds[subject.Name]
	}

	// get a client for the Seed cluster; this uses a restmapper and is therefore cached for better performance
	seedClient, err := v.seedClientGetter(subject)
	if err != nil {
		return fmt.Errorf("failed to get Seed client: %w", err)
	}

	// this can be nil on new seed clusters
	existingSeed := existingSeeds[subject.Name]

	// remove the seed itself from the map, so uniqueness checks won't fail
	delete(existingSeeds, subject.Name)

	// collect datacenter names
	subjectDatacenters := sets.New[string]()
	existingDatacenters := sets.New[string]()

	if !isDelete {
		// this has no effect on the DC uniqueness check, but makes the
		// cluster-remaining-in-DC check easier
		subjectDatacenters = sets.KeySet(subject.Spec.Datacenters)
	}

	// check if the subject introduces a datacenter that already exists
	for _, existing := range existingSeeds {
		datacenters := sets.KeySet(existing.Spec.Datacenters)

		if duplicates := subjectDatacenters.Intersection(datacenters); duplicates.Len() > 0 {
			return fmt.Errorf("Seed redefines existing datacenters %v from Seed %q; datacenter names must be globally unique", sets.List(duplicates), existing.Name)
		}

		existingDatacenters = existingDatacenters.Union(datacenters)
	}

	// check if all DCs have exactly one provider and that the provider
	// is never changed after it has been set once
	for dcName, dc := range subject.Spec.Datacenters {
		providerName, err := kubermaticv1helper.DatacenterCloudProviderName(&dc.Spec)
		if err != nil {
			return fmt.Errorf("datacenter %q is invalid: %w", dcName, err)
		}
		if providerName == "" {
			return fmt.Errorf("datacenter %q has no provider defined", dcName)
		}

		if dc.Spec.Kubevirt != nil {
			if err := validateKubeVirtSupportedOS(dc.Spec.Kubevirt); err != nil {
				return err
			}
		}

		if dc.Spec.Baremetal != nil && dc.Spec.Baremetal.Tinkerbell != nil {
			if err := validateTinkerbellSupportedOS(dc.Spec.Baremetal.Tinkerbell); err != nil {
				return err
			}
		}

		if existingSeed == nil {
			continue
		}

		existingDC, ok := existingSeed.Spec.Datacenters[dcName]
		if !ok {
			continue
		}

		existingProvider, _ := kubermaticv1helper.DatacenterCloudProviderName(&existingDC.Spec)
		if providerName != existingProvider {
			return fmt.Errorf("cannot change datacenter %q provider from %q to %q", dcName, existingProvider, providerName)
		}
	}

	if err := validateNoClustersRemaining(ctx, seedClient, subject, subjectDatacenters, existingDatacenters); err != nil {
		return err
	}

	if err := validateDefaultAPIServerAllowedIPRanges(ctx, subject); err != nil {
		return err
	}

	if err := validateNodePortProxyEnvoyConnectionSettings(subject); err != nil {
		return err
	}

	if err := validateEtcdBackupConfiguration(ctx, seedClient, subject); err != nil {
		return err
	}

	if err := validation.ValidateMeteringConfiguration(subject.Spec.Metering); err != nil {
		return err
	}

	return nil
}

func validateDefaultAPIServerAllowedIPRanges(ctx context.Context, seed *kubermaticv1.Seed) error {
	if len(seed.Spec.DefaultAPIServerAllowedIPRanges) > 0 {
		for _, cidr := range seed.Spec.DefaultAPIServerAllowedIPRanges {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				return fmt.Errorf("%s is not a valid IP range", cidr)
			}
		}
	}
	return nil
}

func validateNodePortProxyEnvoyConnectionSettings(seed *kubermaticv1.Seed) error {
	settings := seed.Spec.NodeportProxy.Envoy.ConnectionSettings

	durationFields := []struct {
		field string
		value time.Duration
	}{
		{
			field: "spec.nodeportProxy.envoy.connectionSettings.sniListenerIdleTimeout",
			value: settings.SNIListenerIdleTimeout.Duration,
		},
		{
			field: "spec.nodeportProxy.envoy.connectionSettings.tunnelingConnectionIdleTimeout",
			value: settings.TunnelingConnectionIdleTimeout.Duration,
		},
		{
			field: "spec.nodeportProxy.envoy.connectionSettings.tunnelingStreamIdleTimeout",
			value: settings.TunnelingStreamIdleTimeout.Duration,
		},
		{
			field: "spec.nodeportProxy.envoy.connectionSettings.downstreamTCPKeepaliveTime",
			value: settings.DownstreamTCPKeepaliveTime.Duration,
		},
		{
			field: "spec.nodeportProxy.envoy.connectionSettings.downstreamTCPKeepaliveInterval",
			value: settings.DownstreamTCPKeepaliveInterval.Duration,
		},
		{
			field: "spec.nodeportProxy.envoy.connectionSettings.upstreamTCPKeepaliveTime",
			value: settings.UpstreamTCPKeepaliveTime.Duration,
		},
		{
			field: "spec.nodeportProxy.envoy.connectionSettings.upstreamTCPKeepaliveInterval",
			value: settings.UpstreamTCPKeepaliveInterval.Duration,
		},
	}

	for _, d := range durationFields {
		if d.value < 0 {
			return fmt.Errorf("%s must be >= 0", d.field)
		}

		if d.value > 0 && d.value < time.Second {
			return fmt.Errorf("%s must be 0 or >= 1s", d.field)
		}
	}

	return nil
}

func validateNoClustersRemaining(ctx context.Context, seedClient ctrlruntimeclient.Client, _ *kubermaticv1.Seed, subjectDatacenters, existingDatacenters sets.Set[string]) error {
	// new seed clusters might not yet have the CRDs installed into them,
	// which for the purpose of this validation is not a problem and simply
	// means there can be no Clusters on that seed
	crd := apiextensionsv1.CustomResourceDefinition{}
	key := types.NamespacedName{Name: "clusters.kubermatic.k8c.io"}
	crdExists := true
	if err := seedClient.Get(ctx, key, &crd); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to probe for Cluster CRD: %w", err)
		}

		crdExists = false
	}

	if crdExists {
		// check if there are still clusters using DCs not defined anymore
		clusters := &kubermaticv1.ClusterList{}
		if err := seedClient.List(ctx, clusters); err != nil {
			return fmt.Errorf("failed to list clusters: %w", err)
		}

		// list of all datacenters after the seed would have been persisted
		finalDatacenters := subjectDatacenters.Union(existingDatacenters)

		for _, cluster := range clusters.Items {
			if !finalDatacenters.Has(cluster.Spec.Cloud.DatacenterName) {
				return fmt.Errorf("datacenter %q is still in use by cluster %q, cannot delete it", cluster.Spec.Cloud.DatacenterName, cluster.Name)
			}
		}
	}

	return nil
}

func validateEtcdBackupConfiguration(ctx context.Context, seedClient ctrlruntimeclient.Client, subject *kubermaticv1.Seed) error {
	if subject.Spec.EtcdBackupRestore != nil {
		if len(subject.Spec.EtcdBackupRestore.Destinations) == 0 {
			return errors.New("invalid etcd backup configuration: must define at least one backup destination")
		}

		if subject.Spec.EtcdBackupRestore.DefaultDestination != "" {
			if _, exists := subject.Spec.EtcdBackupRestore.Destinations[subject.Spec.EtcdBackupRestore.DefaultDestination]; !exists {
				return fmt.Errorf("invalid etcd backup configuration: default destination %q does not exist", subject.Spec.EtcdBackupRestore.DefaultDestination)
			}
		}

		for name, dest := range subject.Spec.EtcdBackupRestore.Destinations {
			if !resourceNameValidator.MatchString(name) {
				return fmt.Errorf("destination name is invalid, must match %s", resourceNameValidator.String())
			}

			if dest.Credentials != nil {
				etcdBackupSecret := corev1.Secret{}
				if err := seedClient.Get(ctx, types.NamespacedName{Name: dest.Credentials.Name,
					Namespace: dest.Credentials.Namespace}, &etcdBackupSecret); err != nil {
					return fmt.Errorf("invalid etcd backup configuration: invalid destination %q credentials %s: %w", name, dest.Credentials.Name, err)
				}
			}
		}
	}

	return nil
}

func validateKubeVirtSupportedOS(datacenterSpec *kubermaticv1.DatacenterSpecKubevirt) error {
	if datacenterSpec != nil && datacenterSpec.Images.HTTP != nil {
		for os := range datacenterSpec.Images.HTTP.OperatingSystems {
			if _, exist := kubermaticv1.SupportedKubeVirtOS[os]; !exist {
				return fmt.Errorf("invalid/not supported operating system specified: %s", os)
			}
		}
	}
	return nil
}

func validateTinkerbellSupportedOS(datacenterSpec *kubermaticv1.DatacenterSpecTinkerbell) error {
	if datacenterSpec != nil && datacenterSpec.Images.HTTP != nil {
		for os := range datacenterSpec.Images.HTTP.OperatingSystems {
			if _, exist := kubermaticv1.SupportedTinkerbellOS[os]; !exist {
				return fmt.Errorf("invalid/not supported operating system specified: %s", os)
			}
		}
	}
	return nil
}
