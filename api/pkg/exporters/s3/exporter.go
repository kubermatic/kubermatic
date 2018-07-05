package s3

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/minio/minio-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const metricsNamespace = "s3"

type s3Exporter struct {
	ObjectsCount            *prometheus.GaugeVec
	ObjectsLastModifiedDate *prometheus.GaugeVec
	ObjectsEmptyCount       *prometheus.GaugeVec
	QuerySuccess            prometheus.Gauge
	bucket                  string
	minioClient             *minio.Client
}

func MustRun(minioClient *minio.Client, bucket string, listenAddress int) error {

	exporter := s3Exporter{}
	exporter.minioClient = minioClient
	exporter.bucket = bucket

	exporter.ObjectsCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "object_count",
		Help:      "The amount of objects",
	}, []string{"prefix"})
	exporter.ObjectsLastModifiedDate = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "objects_last_modified_object_date",
		Help:      "The amount of objects",
	}, []string{"prefix"})
	exporter.ObjectsEmptyCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "objects_empty_count",
		Help:      "The amount of object with a size of zero",
	}, []string{"prefix"})
	exporter.QuerySuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Name:      "query_success",
		Help:      "Whether querying the S3 was successfull",
	})

	registry := prometheus.NewRegistry()
	if err := registry.Register(exporter.ObjectsCount); err != nil {
		return err
	}
	if err := registry.Register(exporter.ObjectsLastModifiedDate); err != nil {
		return err
	}
	if err := registry.Register(exporter.ObjectsEmptyCount); err != nil {
		return err
	}
	if err := registry.Register(exporter.QuerySuccess); err != nil {
		return err
	}

	promHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !exporter.refreshMetrics(w, r) {
			return
		}
		promHandler.ServeHTTP(w, r)
	})
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%v", listenAddress), nil); err != nil {
			glog.Fatalf("Failed to listen: %v")
		}
	}()

	return nil
}

func (e *s3Exporter) refreshMetrics(w http.ResponseWriter, r *http.Request) bool {
	prefix := r.URL.Query().Get("prefix")
	if prefix == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("prefix url arg is required!\n"))
		return false
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	var objects []minio.ObjectInfo
	for listerObject := range e.minioClient.ListObjects(e.bucket, prefix, true, doneCh) {
		if listerObject.Err != nil {
			glog.Errorf("Error on object %s: %v", listerObject.Key, listerObject.Err)
			e.QuerySuccess.Set(float64(1))
			return true
		}
		objects = append(objects, listerObject)
	}

	e.ObjectsCount.With(prometheus.Labels{"prefix": prefix}).Set(float64(len(objects)))
	return true
}
