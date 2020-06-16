// +build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2020 Loodse GmbH

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

package master

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/crd/migrations/util"
	eeprovider "github.com/kubermatic/kubermatic/api/pkg/ee/provider"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	DatacentersFile    string
	DynamicDatacenters bool
	Kubeconfig         *clientcmdapi.Config
}

// MigrationEnabled returns true if at least one migration is enabled.
func (o Options) MigrationEnabled() bool {
	return o.SeedMigrationEnabled()
}

// SeedMigrationEnabled returns true if the datacenters->seed migration is enabled.
func (o Options) SeedMigrationEnabled() bool {
	return o.DatacentersFile != "" && o.DynamicDatacenters
}

// RunAll runs all migrations that should be run inside a master cluster.
func RunAll(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, kubermaticNamespace string, opt Options) error {
	if err := waitForWebhook(ctx, log, client, kubermaticNamespace); err != nil {
		return fmt.Errorf("failed to wait for webhook: %v", err)
	}

	if opt.SeedMigrationEnabled() {
		log.Info("datacenters given and dynamic datacenters enabled, attempting to migrate datacenters to Seeds")
		if err := migrateDatacenters(ctx, log, client, kubermaticNamespace, opt); err != nil {
			return fmt.Errorf("failed to migrate datacenters.yaml: %v", err)
		}
		log.Info("seed migration completed successfully")
	}

	if err := migrateAllDatacenterEmailRestrictions(ctx, log, client, kubermaticNamespace, opt); err != nil {
		return fmt.Errorf("failed to migrate datacenters' email restrictions: %v", err)
	}

	return nil
}

// waitForWebhook waits for the seed validation webhook to be ready, so that
// the migration can successfully create new Seed resources.
func waitForWebhook(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, kubermaticNamespace string) error {
	// wait for the webhook to be ready
	timeout := 30 * time.Second
	endpoint := types.NamespacedName{Namespace: kubermaticNamespace, Name: "seed-webhook"}

	log.Infow("waiting for webhook to be ready...", "webhook", endpoint, "timeout", timeout)
	if err := wait.Poll(500*time.Millisecond, timeout, func() (bool, error) {
		endpoints := &corev1.Endpoints{}
		if err := client.Get(ctx, endpoint, endpoints); err != nil {
			return false, err
		}
		return len(endpoints.Subsets) > 0, nil
	}); err != nil {
		return fmt.Errorf("failed to wait for webhook: %v", err)
	}
	log.Info("webhook is ready")

	return nil
}

// migrateDatacenters creates Seed CRs based on the given datacenters file.
// Seeds are only ever created and never updated/reconciled to match the
// datacenters.yaml, because the validation webhook prevents any modifications
// while the migration is enabled.
func migrateDatacenters(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, kubermaticNamespace string, opt Options) error {
	seeds, err := eeprovider.LoadSeeds(opt.DatacentersFile)
	if err != nil {
		return fmt.Errorf("failed to load %s: %v", opt.DatacentersFile, err)
	}

	for name, seed := range seeds {
		log := log.With("seed", name)
		log.Info("processing Seed...")

		seed.Namespace = kubermaticNamespace

		// create a kubeconfig Secret just for this seed
		objectRef, err := createSeedKubeconfig(ctx, log, client, seed, opt)
		if err != nil {
			return fmt.Errorf("failed to create kubeconfig secret for seed %s: %v", name, err)
		}

		seed.Spec.Kubeconfig = *objectRef

		// create the seed itself
		key, err := ctrlruntimeclient.ObjectKeyFromObject(seed)
		if err != nil {
			return fmt.Errorf("failed to create object key for seed %s: %v", name, err)
		}

		log.Info("checking for Seed existence...")
		existingSeed := kubermaticv1.Seed{}
		err = client.Get(ctx, key, &existingSeed)
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return fmt.Errorf("failed to get Seed %s: %v", name, err)
			}

			log.Info("creating Seed CR...")
			if err := client.Create(ctx, seed); err != nil {
				return fmt.Errorf("failed to create seed %s: %v", name, err)
			}
		}
	}

	return nil
}

// migrateDatacenterEmailRestrictions removes the `requiredEmailDomain` field of DCs and move its value to `requiredEmailDomains`
func migrateAllDatacenterEmailRestrictions(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, kubermaticNamespace string, opt Options) error {
	seedList := &kubermaticv1.SeedList{}
	if err := client.List(ctx, seedList); err != nil {
		return fmt.Errorf("failed to list seeds: %s", err)
	}

	for _, seed := range seedList.Items {
		log := log.With("seed", seed.Name)
		log.Info("processing Seed...")

		anyDCchanged := false
		for dcName, dc := range seed.Spec.Datacenters {
			if dc.Spec.RequiredEmailDomain == "" {
				continue
			}

			if len(dc.Spec.RequiredEmailDomains) > 0 {
				return fmt.Errorf("datacenter %s->%s has both `requiredEmailDomain` and `requiredEmailDomains` set", seed.Name, dcName)
			}

			dc.Spec.RequiredEmailDomains = []string{dc.Spec.RequiredEmailDomain}
			dc.Spec.RequiredEmailDomain = ""
			seed.Spec.Datacenters[dcName] = dc
			anyDCchanged = true
			log.Warnf("datacenter %q is using the deprecated field `requiredEmailDomain` - plese migrate to `requiredEmailDomains` instead", dcName)
		}

		// Update the seed object only if any of the DCs were actually migrated.
		if anyDCchanged {
			if err := client.Update(ctx, &seed); err != nil {
				return fmt.Errorf("failed to update seed %s: %s", seed.Name, err)
			}
		}
	}

	return nil
}

// createSeedKubeconfig creates a new Secret with a kubeconfig contains only the credentials
// required for connecting to the given seed. If the Secret already exists, nothing happens.
func createSeedKubeconfig(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, seed *kubermaticv1.Seed, opt Options) (*corev1.ObjectReference, error) {
	kubeconfig, err := util.SingleSeedKubeconfig(opt.Kubeconfig, seed.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubeconfig: %v", err)
	}

	secret, fieldPath, err := util.CreateKubeconfigSecret(kubeconfig, "kubeconfig-"+seed.Name, seed.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubeconfig Secret: %v", err)
	}

	key, err := ctrlruntimeclient.ObjectKeyFromObject(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create object key: %v", err)
	}

	log.Info("checking for kubeconfig Secret...")
	existingSecret := corev1.Secret{}
	err = client.Get(ctx, key, &existingSecret)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get Secret: %v", err)
		}

		log.Info("creating Secret...")
		if err := client.Create(ctx, secret); err != nil {
			return nil, fmt.Errorf("failed to create Secret: %v", err)
		}
	}

	return &corev1.ObjectReference{
		APIVersion: "v1",
		Kind:       "Secret",
		Name:       secret.Name,
		Namespace:  secret.Namespace,
		FieldPath:  fieldPath,
	}, nil
}
