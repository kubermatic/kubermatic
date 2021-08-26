// +build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Loodse GmbH

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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	k8cerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateOrUpdateCredentials creates or updates the metering tool credentials.
func CreateOrUpdateCredentials(ctx context.Context, request interface{}, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) error {
	if seedsGetter == nil || seedClientGetter == nil {
		return errors.New("parameter seedsGetter nor seedClientGetter cannot be nil")
	}

	req, ok := request.(credentials)
	if !ok {
		return k8cerrors.NewBadRequest("invalid request")
	}

	if err := req.Validate(); err != nil {
		return err
	}

	clients, err := getSeedsClient(seedsGetter, seedClientGetter)
	if err != nil {
		return fmt.Errorf("failed to gety seed clients: %v", err)
	}

	data := map[string][]byte{
		"accessKey": []byte(req.AccessKey),
		"secretKey": []byte(req.SecretKey),
		"bucket":    []byte(req.BucketName),
		"endpoint":  []byte(req.Endpoint),
	}

	for _, client := range clients {
		if err := createOrUpdateMeteringToolSecret(ctx, client, data); err != nil {
			return fmt.Errorf("failed to create or update metering tool credentials: %v", err)
		}
	}

	return nil
}

func getSeedsClient(seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) ([]ctrlruntimeclient.Client, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return nil, fmt.Errorf("failed to get seeds: %v", err)
	}

	var seedClients = make([]ctrlruntimeclient.Client, len(seeds))

	for _, seed := range seeds {
		seedClient, err := seedClientGetter(seed)
		if err != nil {
			return nil, fmt.Errorf("failed to get seed client for seed %q: %v", seed, err)
		}

		seedClients = append(seedClients, seedClient)
	}

	if len(seedClients) < 1 {
		return nil, errors.New("no seeds found")
	}

	return seedClients, nil
}

// credentials contains the aws credentials to access s3 bucket.
type credentials struct {
	BucketName string `json:"bucketName"`
	AccessKey  string `json:"accessKey"`
	SecretKey  string `json:"secretKey"`
	Endpoint   string `json:"endpoint"`
}

func (c credentials) Validate() error {
	if c.Endpoint == "" || c.AccessKey == "" || c.SecretKey == "" || c.BucketName == "" {
		return fmt.Errorf("accessKey, secretKey, bucketName or endpoint cannot be empty")
	}

	return nil
}

func DecodeMeteringToolCredentials(r *http.Request) (interface{}, error) {
	var req credentials
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

// createOrUpdateMeteringToolSecret creates an s3 secret with all the needed aws credentials that are used to interact
// with the buckets where the metering tool exports the metering reports.
func createOrUpdateMeteringToolSecret(ctx context.Context, seedClient ctrlruntimeclient.Client, secretData map[string][]byte) error {
	const secretName = "metering-s3"

	namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: secretName}
	existingSecret := &corev1.Secret{}
	if err := seedClient.Get(ctx, namespacedName, existingSecret); err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to probe for secret %q: %v", secretName, err)
	}

	if existingSecret.Name == "" {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: resources.KubermaticNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: secretData,
		}

		if err := seedClient.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create credential secret: %v", err)
		}
	} else {
		if existingSecret.Data == nil {
			existingSecret.Data = map[string][]byte{}
		}

		requiresUpdate := false

		for k, v := range secretData {
			if !bytes.Equal(v, existingSecret.Data[k]) {
				requiresUpdate = true
				break
			}
		}

		if requiresUpdate {
			existingSecret.Data = secretData
			if err := seedClient.Update(ctx, existingSecret); err != nil {
				return fmt.Errorf("failed to update credential secret: %v", err)
			}
		}
	}

	return nil
}
