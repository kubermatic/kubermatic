//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package metering

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AccessKey  = "accessKey"
	SecretKey  = "secretKey"
	Bucket     = "bucket"
	Endpoint   = "endpoint"
	SecretName = "metering-s3"
)

type configurationReq struct {
	Enabled          bool   `json:"enabled"`
	StorageClassName string `json:"storageClassName"`
	StorageSize      string `json:"storageSize"`
}

func (m configurationReq) Validate() error {
	if m.Enabled {
		if m.StorageClassName == "" || m.StorageSize == "" {
			return errors.New("storageClassName or storageSize cannot be empty when the metering tool is enabled")
		}

		if _, err := resource.ParseQuantity(m.StorageSize); err != nil {
			return fmt.Errorf("inapproperiate storageClass size: %w", err)
		}
	}

	return nil
}

func DecodeMeteringConfigurationsReq(r *http.Request) (interface{}, error) {
	var req configurationReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

// CreateOrUpdateConfigurations creates or updates the metering tool configurations.
func CreateOrUpdateConfigurations(ctx context.Context, request interface{}, seedsGetter provider.SeedsGetter, masterClient ctrlruntimeclient.Client) error {
	req, ok := request.(configurationReq)
	if !ok {
		return utilerrors.NewBadRequest("invalid request")
	}
	err := req.Validate()
	if err != nil {
		return utilerrors.NewBadRequest(err.Error())
	}

	seeds, err := seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seed clients: %w", err)
	}

	for name, seed := range seeds {
		if err := updateSeedMeteringConfiguration(ctx, req, seed, masterClient); err != nil {
			return fmt.Errorf("failed to reconcile metering tool credentials on Seed %s: %w", name, err)
		}
	}

	return nil
}

func updateSeedMeteringConfiguration(ctx context.Context, meteringCfg configurationReq, seed *kubermaticv1.Seed, masterClient ctrlruntimeclient.Client) error {
	if seed.Spec.Metering == nil {
		seed.Spec.Metering = &kubermaticv1.MeteringConfiguration{}
	}
	seed.Spec.Metering.Enabled = meteringCfg.Enabled
	seed.Spec.Metering.StorageClassName = meteringCfg.StorageClassName
	seed.Spec.Metering.StorageSize = meteringCfg.StorageSize

	if err := masterClient.Update(ctx, seed); err != nil {
		return fmt.Errorf("failed to update seed %q: %w", seed.Name, err)
	}

	return nil
}

// credentialReq contains the s3 secrets to access s3 bucket.
type credentialReq struct {
	BucketName string `json:"bucketName"`
	AccessKey  string `json:"accessKey"`
	SecretKey  string `json:"secretKey"`
	Endpoint   string `json:"endpoint"`
}

func (c credentialReq) Validate() error {
	if c.Endpoint == "" || c.AccessKey == "" || c.SecretKey == "" || c.BucketName == "" {
		return fmt.Errorf("accessKey, secretKey, bucketName or endpoint cannot be empty")
	}

	return nil
}

func DecodeMeteringSecretReq(r *http.Request) (interface{}, error) {
	var req credentialReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

// CreateOrUpdateCredentials creates or updates the metering tool credentials.
func CreateOrUpdateCredentials(ctx context.Context, request interface{}, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) error {
	if seedsGetter == nil || seedClientGetter == nil {
		return errors.New("parameter seedsGetter nor seedClientGetter must not be nil")
	}

	req, ok := request.(credentialReq)
	if !ok {
		return utilerrors.NewBadRequest("invalid request")
	}

	if err := req.Validate(); err != nil {
		return err
	}

	seeds, err := seedsGetter()
	if err != nil {
		return fmt.Errorf("failed to get seed clients: %w", err)
	}

	data := map[string][]byte{
		AccessKey: []byte(req.AccessKey),
		SecretKey: []byte(req.SecretKey),
		Bucket:    []byte(req.BucketName),
		Endpoint:  []byte(req.Endpoint),
	}

	for seedName, seed := range seeds {
		client, err := seedClientGetter(seed)
		if err != nil {
			return fmt.Errorf("failed to get client for seed %s: %w", seedName, err)
		}

		if err := ensureMeteringToolSecret(ctx, client, seed, data); err != nil {
			return fmt.Errorf("failed to ensure metering tool credentials on seed %s: %w", seedName, err)
		}
	}

	return nil
}

func ensureMeteringToolSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, seed *kubermaticv1.Seed, secretData map[string][]byte) error {
	creator := func() (name string, create reconciling.SecretCreator) {
		return SecretName, func(existing *corev1.Secret) (*corev1.Secret, error) {
			existing.Data = secretData
			return existing, nil
		}
	}

	if err := reconciling.ReconcileSecrets(ctx, []reconciling.NamedSecretCreatorGetter{creator}, seed.Namespace, seedClient); err != nil {
		return err
	}

	return nil
}
