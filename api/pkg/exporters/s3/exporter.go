package s3

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"

	"github.com/golang/glog"
	"github.com/minio/minio-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const metricsNamespace = "s3"

type s3Exporter struct {
	ObjectsCount            *prometheus.GaugeVec
	ObjectsLastModifiedDate *prometheus.GaugeVec
	ObjectsEmptyCount       *prometheus.GaugeVec
	QuerySuccess            prometheus.Gauge
	kubermaticClient        kubermaticclientset.Interface
	bucket                  string
	minioClient             *minio.Client
}

// MustRun starts a s3 exporter or panic
func MustRun(minioClient *minio.Client, kubermaticClient kubermaticclientset.Interface, bucket, listenAddress string) {

	exporter := s3Exporter{}
	exporter.minioClient = minioClient
	exporter.kubermaticClient = kubermaticClient
	exporter.bucket = bucket

	exporter.ObjectsCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "object_count",
		Help:      "The amount of objects",
	}, []string{"cluster"})
	exporter.ObjectsLastModifiedDate = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "object_last_modified_object_time_seconds",
		Help:      "The amount of objects",
	}, []string{"cluster"})
	exporter.ObjectsEmptyCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "empty_object_count",
		Help:      "The amount of object with a size of zero",
	}, []string{"cluster"})
	exporter.QuerySuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "query_success",
		Help:      "Whether querying the S3 was successful",
	})

	registry := prometheus.NewRegistry()
	if err := registry.Register(exporter.ObjectsCount); err != nil {
		glog.Fatal(err)
	}
	if err := registry.Register(exporter.ObjectsLastModifiedDate); err != nil {
		glog.Fatal(err)
	}
	if err := registry.Register(exporter.ObjectsEmptyCount); err != nil {
		glog.Fatal(err)
	}
	if err := registry.Register(exporter.QuerySuccess); err != nil {
		glog.Fatal(err)
	}

	promHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		exporter.refreshMetrics(r)
		promHandler.ServeHTTP(w, r)
	})
	go func() {
		if err := http.ListenAndServe(listenAddress, nil); err != nil {
			glog.Fatalf("Failed to listen: %v", err)
		}
	}()
}

func (e *s3Exporter) refreshMetrics(r *http.Request) {
	clusters, err := e.kubermaticClient.KubermaticV1().Clusters().List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("Failed to list clusters: %v", err)
		e.QuerySuccess.Set(float64(1))
		return
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	var objects []minio.ObjectInfo
	for listerObject := range e.minioClient.ListObjects(e.bucket, "", true, doneCh) {
		if listerObject.Err != nil {
			glog.Errorf("Error on object %s: %v", listerObject.Key, listerObject.Err)
			e.QuerySuccess.Set(float64(1))
			return
		}
		objects = append(objects, listerObject)
	}

	for _, cluster := range clusters.Items {
		e.setMetricsForCluster(objects, cluster.Name)
	}

	return
}

func (e *s3Exporter) setMetricsForCluster(allObjects []minio.ObjectInfo, clusterName string) {

	var clusterObjects []minio.ObjectInfo
	for _, object := range allObjects {
		if strings.HasPrefix(object.Key, fmt.Sprintf("%s-", clusterName)) {
			clusterObjects = append(clusterObjects, object)
		}
	}

	labels := prometheus.Labels{"cluster": clusterName}
	e.ObjectsCount.With(labels).Set(float64(len(clusterObjects)))
	e.ObjectsLastModifiedDate.With(labels).Set(float64(getLastModifiedTimestamp(clusterObjects).UnixNano()))
	e.ObjectsEmptyCount.With(labels).Set(float64(getEmptyObjectCount(clusterObjects)))
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
