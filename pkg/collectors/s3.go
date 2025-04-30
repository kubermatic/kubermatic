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

package collectors

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type s3Collector struct {
	ObjectCount            *prometheus.Desc
	ObjectLastModifiedDate *prometheus.Desc
	EmptyObjectCount       *prometheus.Desc
	QuerySuccess           *prometheus.Desc
	client                 ctrlruntimeclient.Reader
	bucket                 string
	minioClient            *minio.Client
	logger                 *zap.SugaredLogger
}

// MustRegisterS3Collector registers the S3 collector.
func MustRegisterS3Collector(minioClient *minio.Client, client ctrlruntimeclient.Reader, bucket string, logger *zap.SugaredLogger) {
	collector := s3Collector{}
	collector.minioClient = minioClient
	collector.client = client
	collector.bucket = bucket
	collector.logger = logger

	collector.ObjectCount = prometheus.NewDesc(
		"kubermatic_s3_object_count",
		"The amount of objects partitioned by cluster",
		[]string{"cluster"}, nil)
	collector.ObjectLastModifiedDate = prometheus.NewDesc(
		"kubermatic_s3_object_last_modified_time_seconds",
		"Modification time of the last modified object",
		[]string{"cluster"}, nil)
	collector.EmptyObjectCount = prometheus.NewDesc(
		"kubermatic_s3_empty_object_count",
		"The amount of empty objects (size=0) partitioned by cluster",
		[]string{"cluster"}, nil)
	collector.QuerySuccess = prometheus.NewDesc(
		"kubermatic_s3_query_success",
		"Whether querying the S3 was successful",
		nil, nil)

	prometheus.MustRegister(&collector)
}

func (e *s3Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.ObjectCount
	ch <- e.ObjectLastModifiedDate
	ch <- e.EmptyObjectCount
	ch <- e.QuerySuccess
}

func (e *s3Collector) Collect(ch chan<- prometheus.Metric) {
	clusterList := &kubermaticv1.ClusterList{}
	if err := e.client.List(context.Background(), clusterList); err != nil {
		e.logger.Errorw("Failed to list clusters", zap.Error(err))
		ch <- prometheus.MustNewConstMetric(
			e.QuerySuccess,
			prometheus.GaugeValue,
			float64(1))
		return
	}

	logger := e.logger.With("bucket", e.bucket)
	listOpts := minio.ListObjectsOptions{
		Recursive: true,
	}

	var objects []minio.ObjectInfo
	for listerObject := range e.minioClient.ListObjects(context.Background(), e.bucket, listOpts) {
		if listerObject.Err != nil {
			logger.Errorw("Error on object", "object", listerObject.Key, zap.Error(listerObject.Err))
			ch <- prometheus.MustNewConstMetric(
				e.QuerySuccess,
				prometheus.GaugeValue,
				float64(1))
			return
		}
		objects = append(objects, listerObject)
	}

	for _, cluster := range clusterList.Items {
		e.setMetricsForCluster(ch, objects, cluster.Name)
	}
}

func (e *s3Collector) setMetricsForCluster(ch chan<- prometheus.Metric, allObjects []minio.ObjectInfo, clusterName string) {
	var clusterObjects []minio.ObjectInfo
	for _, object := range allObjects {
		if strings.HasPrefix(object.Key, fmt.Sprintf("%s-", clusterName)) {
			clusterObjects = append(clusterObjects, object)
		}
	}

	ch <- prometheus.MustNewConstMetric(
		e.ObjectCount,
		prometheus.GaugeValue,
		float64(len(clusterObjects)),
		clusterName)
	ch <- prometheus.MustNewConstMetric(
		e.ObjectLastModifiedDate,
		prometheus.GaugeValue,
		float64(getLastModifiedTimestamp(clusterObjects).UnixNano()),
		clusterName)
	ch <- prometheus.MustNewConstMetric(
		e.EmptyObjectCount,
		prometheus.GaugeValue,
		float64(getEmptyObjectCount(clusterObjects)),
		clusterName)
}
