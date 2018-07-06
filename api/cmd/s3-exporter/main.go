package main

import (
	"flag"
	"os"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/exporters/s3"

	"github.com/golang/glog"
	"github.com/minio/minio-go"
)

func main() {
	endpointWithProto := flag.String("endpoint", "", "https://my-s3.com:9000")
	accessKeyID := flag.String("access-key-id", "", "S3 Access key")
	secretAccessKey := flag.String("secret-access-key", "", "S3 Secret Access Key")
	bucket := flag.String("bucket", "kubermatic-etcd-backups", "The bucket to monitor")
	listenPort := flag.Int("listen-port", 9340, "The port to listen on")
	flag.Parse()

	if *accessKeyID == "" {
		*accessKeyID = os.Getenv("ACCESS_KEY_ID")
	}
	if *secretAccessKey == "" {
		*secretAccessKey = os.Getenv("SECRET_ACCESS_KEY")
	}

	if *endpointWithProto == "" || *accessKeyID == "" || *secretAccessKey == "" {
		glog.Fatalf("All of 'endpoint', 'access-key-id' and 'secret-access-key' must be set!")
	}

	secure := true
	if strings.HasPrefix(*endpointWithProto, "http://") {
		glog.Info("Disabling tls due to http:// prefix in url..")
		secure = false
	}
	endpoint := strings.TrimPrefix(*endpointWithProto, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	stopChannel := make(chan struct{})
	minioClient, err := minio.New(endpoint, *accessKeyID, *secretAccessKey, secure)
	if err != nil {
		glog.Fatalf("Failed to get S3 client: %v", err)
	}

	s3.MustRun(minioClient, *bucket, *listenPort)

	glog.Infof("Successfully started, listening on port %v", *listenPort)
	<-stopChannel
	glog.Infof("Shutting down..")
}
