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

package collectors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	etcdbackup "k8c.io/kubermatic/v2/pkg/resources/etcd/backup"
	"k8c.io/kubermatic/v2/pkg/util/s3"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type clusterBackupCollector struct {
	ObjectCount            *prometheus.Desc
	ObjectLastModifiedDate *prometheus.Desc
	EmptyObjectCount       *prometheus.Desc
	QuerySuccess           *prometheus.Desc
	client                 ctrlruntimeclient.Reader
	logger                 *zap.SugaredLogger
	caBundle               *certificates.CABundle
	seedGetter             provider.SeedGetter
}

// MustRegisterClusterBackupCollector registers the cluster backup collector.
func MustRegisterClusterBackupCollector(
	registry prometheus.Registerer,
	client ctrlruntimeclient.Reader,
	logger *zap.SugaredLogger,
	caBundle *certificates.CABundle,
	seedGetter provider.SeedGetter,
) {
	collector := clusterBackupCollector{}
	collector.client = client
	collector.logger = logger
	collector.caBundle = caBundle
	collector.seedGetter = seedGetter

	collector.ObjectCount = prometheus.NewDesc(
		"kubermatic_etcdbackup_object_count",
		"The amount of objects partitioned by backup destination and cluster",
		[]string{"destination", "cluster"}, nil)
	collector.ObjectLastModifiedDate = prometheus.NewDesc(
		"kubermatic_etcdbackup_object_last_modified_time_seconds",
		"Modification time of the last modified object",
		[]string{"destination", "cluster"}, nil)
	collector.EmptyObjectCount = prometheus.NewDesc(
		"kubermatic_etcdbackup_empty_object_count",
		"The amount of empty objects (size=0) partitioned by backup destination and cluster",
		[]string{"destination", "cluster"}, nil)
	collector.QuerySuccess = prometheus.NewDesc(
		"kubermatic_etcdbackup_query_success",
		"Whether querying the S3 was successful",
		[]string{"destination"}, nil)

	registry.MustRegister(&collector)
}

func (c *clusterBackupCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.ObjectCount
	ch <- c.ObjectLastModifiedDate
	ch <- c.EmptyObjectCount
	ch <- c.QuerySuccess
}

func (c *clusterBackupCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.collect(context.Background(), ch); err != nil {
		c.logger.Errorw("Failed to collect metrics", zap.Error(err))
	}
}

func (c *clusterBackupCollector) collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	seed, err := c.seedGetter()
	if err != nil {
		return err
	}

	// For the legacy backup mechanism, the S3 credentials
	// are never configured directly in KKP, instead the admin
	// only configures full BackupContainerSpecs.
	// Because of that, this collector can only work with the
	// new backup/restore mechanism.
	if !seed.IsEtcdAutomaticBackupEnabled() {
		return nil
	}

	clusterList := &kubermaticv1.ClusterList{}
	if err := c.client.List(ctx, clusterList); err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	for destName, destination := range seed.Spec.EtcdBackupRestore.Destinations {
		logger := c.logger.With("destination", destName)
		logger.Debug("Collecting metrics")

		success := float64(1)

		if err := c.collectDestination(ctx, ch, clusterList.Items, destName, destination); err != nil {
			// do not return an error, but try to keep gathering data for the other destinations
			logger.Errorw("Failed to collect metrics for backup destination", zap.Error(err))
			success = 0
		}

		ch <- prometheus.MustNewConstMetric(c.QuerySuccess, prometheus.GaugeValue, success, destName)
	}

	return nil
}

func (c *clusterBackupCollector) collectDestination(ctx context.Context, ch chan<- prometheus.Metric, clusters []kubermaticv1.Cluster, destName string, destination *kubermaticv1.BackupDestination) error {
	listOpts := minio.ListObjectsOptions{
		Recursive: true,
	}

	s3Client, err := c.getS3Client(ctx, destination)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	var objects []minio.ObjectInfo
	for listerObject := range s3Client.ListObjects(ctx, destination.BucketName, listOpts) {
		if listerObject.Err != nil {
			return fmt.Errorf("failed to list objects in bucket: %w", listerObject.Err)
		}

		objects = append(objects, listerObject)
	}

	for _, cluster := range clusters {
		c.setMetricsForCluster(ch, destination, objects, destName, cluster.Name)
	}

	return nil
}

func (c *clusterBackupCollector) setMetricsForCluster(ch chan<- prometheus.Metric, _ *kubermaticv1.BackupDestination, allObjects []minio.ObjectInfo, destName string, clusterName string) {
	var clusterObjects []minio.ObjectInfo
	for _, object := range allObjects {
		if strings.HasPrefix(object.Key, fmt.Sprintf("%s-", clusterName)) {
			clusterObjects = append(clusterObjects, object)
		}
	}

	labelValues := []string{destName, clusterName}

	lastModTimestamp := int64(0)
	if lastMod := getLastModifiedTimestamp(clusterObjects); !lastMod.IsZero() {
		lastModTimestamp = lastMod.Unix()
	}

	ch <- prometheus.MustNewConstMetric(c.ObjectCount, prometheus.GaugeValue, float64(len(clusterObjects)), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.ObjectLastModifiedDate, prometheus.GaugeValue, float64(lastModTimestamp), labelValues...)
	ch <- prometheus.MustNewConstMetric(c.EmptyObjectCount, prometheus.GaugeValue, float64(getEmptyObjectCount(clusterObjects)), labelValues...)
}

func (c *clusterBackupCollector) getS3Client(ctx context.Context, destination *kubermaticv1.BackupDestination) (*minio.Client, error) {
	if destination.Credentials == nil {
		return nil, fmt.Errorf("credentials not set for backup destination %q", destination)
	}

	key := types.NamespacedName{
		Name:      destination.Credentials.Name,
		Namespace: destination.Credentials.Namespace,
	}

	creds := &corev1.Secret{}
	if err := c.client.Get(ctx, key, creds); err != nil {
		return nil, fmt.Errorf("failed to retrieve credentials secret: %w", err)
	}

	accessKey := string(creds.Data[etcdbackup.AccessKeyIDEnvVarKey])
	secretKey := string(creds.Data[etcdbackup.SecretAccessKeyEnvVarKey])
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("backup credentials do not contain %q or %q keys", etcdbackup.AccessKeyIDEnvVarKey, etcdbackup.SecretAccessKeyEnvVarKey)
	}

	return s3.NewClient(destination.Endpoint, accessKey, secretKey, c.caBundle.CertPool())
}

func getLastModifiedTimestamp(objects []minio.ObjectInfo) (lastmodifiedTimestamp time.Time) {
	for _, object := range objects {
		if object.LastModified.After(lastmodifiedTimestamp) {
			lastmodifiedTimestamp = object.LastModified
		}
	}

	return lastmodifiedTimestamp
}

func getEmptyObjectCount(objects []minio.ObjectInfo) (emptyObjects int) {
	for _, object := range objects {
		if object.Size == 0 {
			emptyObjects++
		}
	}

	return emptyObjects
}
