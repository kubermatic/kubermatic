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

// FetchClusterBackupConfigWithSeedClient returns the Cluster Backup configuration using a seed client to access the seed object.
func FetchClusterBackupConfigWithSeedClient(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *v1.Cluster, log *zap.SugaredLogger) (*resources.ClusterBackupConfig, error) {
	if !cluster.Spec.Features[v1.ClusterFeatureUserClusterBackup] {
		return &resources.ClusterBackupConfig{Enabled: false}, nil
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
	return FetchClusterBackupConfig(ctx, seed, cluster, log)
}

// FetchClusterBackupConfig returns the Cluster Backup configuration from the seed object directly
func FetchClusterBackupConfig(ctx context.Context, seed *v1.Seed, cluster *v1.Cluster, log *zap.SugaredLogger) (*resources.ClusterBackupConfig, error) {
	if !cluster.Spec.Features[v1.ClusterFeatureUserClusterBackup] {
		return &resources.ClusterBackupConfig{Enabled: false}, nil
	}

	// We pick the default backup destination for now. This behavior will change once we add the API.
	destinations := seed.Spec.EtcdBackupRestore.Destinations
	defaultDestination := seed.Spec.EtcdBackupRestore.DefaultDestination
	if len(destinations) == 0 || defaultDestination == "" {
		log.Infof("seed [%s] has no backup destinations or no default backup destinations defined. Skipping cluster backup config for cluster [%s]", seed.Name, cluster.Name)
		return &resources.ClusterBackupConfig{Enabled: false}, nil
	}
	dest, ok := destinations[defaultDestination]
	if !ok {
		return nil, fmt.Errorf("configured default destination [%s] doesn't exist", defaultDestination)
	}
	if dest.BucketName == "" || dest.Endpoint == "" || dest.Credentials == nil {
		return nil, fmt.Errorf("failed to validate backup destination configuration: bucketName, endpoint or credentials are not valid")
	}
	return &resources.ClusterBackupConfig{
		Enabled:     cluster.Spec.Features[v1.ClusterFeatureUserClusterBackup],
		Destination: dest,
	}, nil
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
