package master

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kubeconfigFieldPath = "kubeconfig"
)

type MigrationOptions struct {
	DatacentersFile    string
	DynamicDatacenters bool
	Kubeconfig         *clientcmdapi.Config
}

// MigrationEnabled returns true if at least one migration is enabled.
func (o MigrationOptions) MigrationEnabled() bool {
	return o.SeedMigrationEnabled()
}

// SeedMigrationEnabled returns true if the datacenters->seed migration is enabled.
func (o MigrationOptions) SeedMigrationEnabled() bool {
	return o.DatacentersFile != "" && o.DynamicDatacenters
}

// RunAll runs all migrations that should be run inside a master cluster.
func RunAll(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, kubermaticNamespace string, opt MigrationOptions) error {
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
func migrateDatacenters(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, kubermaticNamespace string, opt MigrationOptions) error {
	seeds, err := provider.LoadSeeds(opt.DatacentersFile)
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
func migrateAllDatacenterEmailRestrictions(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, kubermaticNamespace string, opt MigrationOptions) error {
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
			log.Warn("datacenter %q is using the deprecated field `requiredEmailDomain` - plese migrate to `requiredEmailDomains` instead", dcName)
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
func createSeedKubeconfig(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, seed *kubermaticv1.Seed, opt MigrationOptions) (*corev1.ObjectReference, error) {
	kubeconfig, err := singleSeedKubeconfig(seed, opt.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubeconfig: %v", err)
	}

	encoded, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize kubeconfig: %v", err)
	}

	secret := &corev1.Secret{}
	secret.Name = "kubeconfig-" + seed.Name
	secret.Namespace = seed.Namespace
	secret.Data = map[string][]byte{
		kubeconfigFieldPath: encoded,
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
		FieldPath:  kubeconfigFieldPath,
	}, nil
}

// singleSeedKubeconfig takes a kubeconfig and returns a new kubeconfig that only has the
// required parts to connect to the given seed. An error is returned when the given seed
// has no valid matching context in the kubeconfig.
func singleSeedKubeconfig(seed *kubermaticv1.Seed, kubeconfig *clientcmdapi.Config) (*clientcmdapi.Config, error) {
	if kubeconfig == nil {
		return nil, errors.New("no kubeconfig defined")
	}

	contextName := seed.Name

	context, exists := kubeconfig.Contexts[contextName]
	if !exists {
		return nil, fmt.Errorf("no context named %s found in kubeconfig", contextName)
	}
	clusterName := context.Cluster
	authName := context.AuthInfo

	cluster, exists := kubeconfig.Clusters[clusterName]
	if !exists {
		return nil, fmt.Errorf("no cluster named %s found in kubeconfig", clusterName)
	}

	auth, exists := kubeconfig.AuthInfos[authName]
	if !exists {
		return nil, fmt.Errorf("no user named %s found in kubeconfig", authName)
	}

	config := clientcmdapi.NewConfig()
	config.CurrentContext = contextName
	config.Contexts[contextName] = context
	config.Clusters[clusterName] = cluster
	config.AuthInfos[authName] = auth

	return config, nil
}
