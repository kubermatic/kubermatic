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

package s3

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type s3Exporter struct {
	ObjectCount            *prometheus.Desc
	ObjectLastModifiedDate *prometheus.Desc
	EmptyObjectCount       *prometheus.Desc
	QuerySuccess           *prometheus.Desc
	kubermaticClient       kubermaticclientset.Interface
	bucket                 string
	minioClient            *minio.Client
	logger                 *zap.SugaredLogger
}

// MustRun starts a s3 exporter or panic
func MustRun(minioClient *minio.Client, kubermaticClient kubermaticclientset.Interface, bucket, listenAddress string, logger *zap.SugaredLogger) {
	exporter := s3Exporter{}
	exporter.minioClient = minioClient
	exporter.kubermaticClient = kubermaticClient
	exporter.bucket = bucket
	exporter.logger = logger

	exporter.ObjectCount = prometheus.NewDesc(
		"kubermatic_s3_object_count",
		"The amount of objects partitioned by cluster",
		[]string{"cluster"}, nil)
	exporter.ObjectLastModifiedDate = prometheus.NewDesc(
		"kubermatic_s3_object_last_modified_time_seconds",
		"Modification time of the last modified object",
		[]string{"cluster"}, nil)
	exporter.EmptyObjectCount = prometheus.NewDesc(
		"kubermatic_s3_empty_object_count",
		"The amount of empty objects (size=0) partitioned by cluster",
		[]string{"cluster"}, nil)
	exporter.QuerySuccess = prometheus.NewDesc(
		"kubermatic_s3_query_success",
		"Whether querying the S3 was successful",
		nil, nil)

	prometheus.MustRegister(&exporter)

	http.Handle("/", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(listenAddress, nil); err != nil {
			logger.Fatalw("Failed to listen", zap.Error(err))
		}
	}()
}

func (e *s3Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.ObjectCount
	ch <- e.ObjectLastModifiedDate
	ch <- e.EmptyObjectCount
	ch <- e.QuerySuccess
}

func (e *s3Exporter) Collect(ch chan<- prometheus.Metric) {
	clusters, err := e.kubermaticClient.KubermaticV1().Clusters().List(metav1.ListOptions{})
	if err != nil {
		e.logger.Errorw("Failed to list clusters", zap.Error(err))
		ch <- prometheus.MustNewConstMetric(
			e.QuerySuccess,
			prometheus.GaugeValue,
			float64(1))
		return
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	logger := e.logger.With("bucket", e.bucket)

	var objects []minio.ObjectInfo
	for listerObject := range e.minioClient.ListObjects(e.bucket, "", true, doneCh) {
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

	for _, cluster := range clusters.Items {
		e.setMetricsForCluster(ch, objects, cluster.Name)
	}
}

func (e *s3Exporter) setMetricsForCluster(ch chan<- prometheus.Metric, allObjects []minio.ObjectInfo, clusterName string) {
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
