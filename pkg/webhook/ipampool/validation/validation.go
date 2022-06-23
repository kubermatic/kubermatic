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
	"math"
	"net"
	"net/http"
	"strings"

	"github.com/go-logr/logr"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	admissionv1 "k8s.io/api/admission/v1"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler for validating Kubermatic IPAMPool CRD.
type AdmissionHandler struct {
	log              logr.Logger
	decoder          *admission.Decoder
	seedGetter       provider.SeedGetter
	seedClientGetter provider.SeedClientGetter
}

// NewAdmissionHandler returns a new IPAMPool AdmissionHandler.
func NewAdmissionHandler(seedGetter provider.SeedGetter, seedClientGetter provider.SeedClientGetter) *AdmissionHandler {
	return &AdmissionHandler{
		seedGetter:       seedGetter,
		seedClientGetter: seedClientGetter,
	}
}

func (h *AdmissionHandler) SetupWebhookWithManager(mgr ctrlruntime.Manager) {
	mgr.GetWebhookServer().Register("/validate-kubermatic-k8c-io-v1-ipampool", &webhook.Admission{Handler: h})
}

func (h *AdmissionHandler) InjectLogger(l logr.Logger) error {
	h.log = l.WithName("ipampool-validation-handler")
	return nil
}

func (h *AdmissionHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *AdmissionHandler) Handle(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	ipamPool := &kubermaticv1.IPAMPool{}

	switch req.Operation {
	case admissionv1.Create:
		if err := h.decoder.Decode(req, ipamPool); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if err := validateIPAMPool(ipamPool); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("IPAMPool is not valid: %w", err))
		}

	case admissionv1.Update:
		if err := h.decoder.Decode(req, ipamPool); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if err := validateIPAMPool(ipamPool); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("IPAMPool is not valid: %w", err))
		}

		oldIPAMPool := &kubermaticv1.IPAMPool{}
		if err := h.decoder.DecodeRaw(req.OldObject, oldIPAMPool); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		err := h.validateUpdate(ctx, oldIPAMPool, ipamPool)
		if err != nil {
			h.log.Info("IPAMPool validation failed", "error", err)
			return webhook.Denied(fmt.Sprintf("IPAMPool update request %s cannot happen: %v", req.UID, err))
		}

	case admissionv1.Delete:
		if err := h.decoder.Decode(req, ipamPool); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		err := h.validateDelete(ctx, ipamPool, "")
		if err != nil {
			h.log.Info("IPAMPool deletion failed", "error", err)
			return webhook.Denied(fmt.Sprintf("IPAMPool deletion request %s cannot happen: %v", req.UID, err))
		}

	default:
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("%s not supported on IPAMPool resources", req.Operation))
	}

	return webhook.Allowed(fmt.Sprintf("IPAMPool validation request %s allowed", req.UID))
}

func validateIPAMPool(ipamPool *kubermaticv1.IPAMPool) error {
	for _, dcConfig := range ipamPool.Spec.Datacenters {
		_, poolSubnet, err := net.ParseCIDR(string(dcConfig.PoolCIDR))
		if err != nil {
			return err
		}
		poolPrefix, bits := poolSubnet.Mask.Size()

		if (bits - poolPrefix) >= 64 {
			return errors.New("the pool is too big to be processed")
		}

		switch dcConfig.Type {
		case kubermaticv1.IPAMPoolAllocationTypeRange:
			numberOfPoolSubnetIPs := uint64(math.Pow(2, float64(bits-poolPrefix)))
			if dcConfig.AllocationRange > numberOfPoolSubnetIPs {
				return errors.New("allocation range cannot be greater than the pool subnet possible number of IP addresses")
			}
		case kubermaticv1.IPAMPoolAllocationTypePrefix:
			if int(dcConfig.AllocationPrefix) < poolPrefix {
				return errors.New("allocation prefix cannot be smaller than the pool subnet mask size")
			}
			if int(dcConfig.AllocationPrefix) > bits {
				return errors.New("invalid allocation prefix for IP version")
			}
		}
	}

	return nil
}

func (h *AdmissionHandler) validateUpdate(ctx context.Context, oldIPAMPool *kubermaticv1.IPAMPool, newIPAMPool *kubermaticv1.IPAMPool) error {
	// loop old IPAMPool datacenters
	for dc, dcOldConfig := range oldIPAMPool.Spec.Datacenters {
		dcNewConfig, dcExistsInNewPool := newIPAMPool.Spec.Datacenters[dc]
		if !dcExistsInNewPool {
			err := h.validateDelete(ctx, oldIPAMPool, dc)
			if err != nil {
				return err
			}
			continue
		}

		if dcOldConfig.PoolCIDR != dcNewConfig.PoolCIDR {
			return errors.New("it's not allowed to update the pool CIDR for a datacenter")
		}

		if dcOldConfig.Type != dcNewConfig.Type {
			return errors.New("it's not allowed to update the allocation type for a datacenter")
		}
	}

	return nil
}

func (h *AdmissionHandler) validateDelete(ctx context.Context, ipamPool *kubermaticv1.IPAMPool, dc string) error {
	client, err := h.getSeedClient(ctx)
	if err != nil {
		return err
	}

	// List all IPAM allocations
	ipamAllocationList := &kubermaticv1.IPAMAllocationList{}
	err = client.List(ctx, ipamAllocationList)
	if err != nil {
		return fmt.Errorf("failed to list IPAM allocations: %w", err)
	}

	// Iterate current IPAM allocations to check if there is an allocation for the pool to be deleted
	var ipamAllocationsNamespaces []string
	for _, ipamAllocation := range ipamAllocationList.Items {
		if ipamAllocation.Name == ipamPool.Name && (dc == "" || ipamAllocation.Spec.DC == dc) {
			ipamAllocationsNamespaces = append(ipamAllocationsNamespaces, ipamAllocation.Namespace)
		}
	}

	if len(ipamAllocationsNamespaces) > 0 {
		return fmt.Errorf("cannot delete IPAMPool for some datacenter because there is existing IPAMAllocation in namespaces (%s)", strings.Join(ipamAllocationsNamespaces, ", "))
	}

	return nil
}

func (h *AdmissionHandler) getSeedClient(ctx context.Context) (ctrlruntimeclient.Client, error) {
	seed, err := h.seedGetter()
	if err != nil {
		return nil, fmt.Errorf("failed to get current seed: %w", err)
	}
	if seed == nil {
		return nil, errors.New("webhook not configured for a seed cluster")
	}

	client, err := h.seedClientGetter(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to get seed client: %w", err)
	}

	return client, nil
}
