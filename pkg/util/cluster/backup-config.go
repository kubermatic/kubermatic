package cluster

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"go.uber.org/zap"
	v1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func FetchClusterBackupConfig(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *v1.Cluster, log *zap.SugaredLogger) (*resources.ClusterBackupConfig, error) {
	clusterBackupConfig := &resources.ClusterBackupConfig{
		Enabled: cluster.Spec.Features[v1.ClusterFeatureUserClusterBackup],
	}
	if !clusterBackupConfig.Enabled {
		return nil, nil
	}
	seedName, err := extractClusterSeedName(cluster.Name, cluster.Status.Address.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract cluster Seed name: %w", err)
	}
	seed := &v1.Seed{}

	if err := seedClient.Get(ctx,
		types.NamespacedName{
			Namespace: resources.KubermaticNamespace,
			Name:      seedName},
		seed); err != nil {
		return nil, fmt.Errorf("failed to get seed [%s]: %w", seedName, err)
	}

	// We pick the default backup destination for now. This behavior will change once we add the API.
	destinations := seed.Spec.EtcdBackupRestore.Destinations
	defaultDestination := seed.Spec.EtcdBackupRestore.DefaultDestination
	if len(destinations) == 0 || defaultDestination != "" {
		log.Infof("seed [%s] has no backup destinations or no default backup destinations defined. Skipping cluster backup config for cluster [%s]", seedName, cluster.Name)
		return nil, nil
	}

	clusterBackupConfig.Enabled = true
	clusterBackupConfig.Destination = destinations[defaultDestination]
	return clusterBackupConfig, nil
}

func extractClusterSeedName(clusterName, clusterURL string) (string, error) {
	u, err := url.Parse(clusterURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse cluster URL: %w", err)
	}
	parts := strings.Split(u.Host, ".")
	if len(parts) < 4 || clusterName != parts[0] { // at least a cluster name, seed name and a base domain.
		return "", fmt.Errorf("invalid cluster URL: %s", u.Host)
	}
	return parts[1], nil
}
