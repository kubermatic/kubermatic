package main

import (
	"flag"
	"os"
	"strings"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/exporters/s3"

	"github.com/golang/glog"
	"github.com/minio/minio-go"

	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	endpointWithProto := flag.String("endpoint", "", "The s3 endpoint, e.G. https://my-s3.com:9000")
	accessKeyID := flag.String("access-key-id", "", "S3 Access key, defaults to the ACCESS_KEY_ID environment variable")
	secretAccessKey := flag.String("secret-access-key", "", "S3 Secret Access Key, defaults to the SECRET_ACCESS_KEY evnironment variable")
	bucket := flag.String("bucket", "kubermatic-etcd-backups", "The bucket to monitor")
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	listenAddress := flag.String("address", ":9340", "The port to listen on")
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

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}
	kubermaticClient := kubermaticclientset.NewForConfigOrDie(config)

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

	s3.MustRun(minioClient, kubermaticClient, *bucket, *listenAddress)

	glog.Infof("Successfully started, listening on %s", *listenAddress)
	<-stopChannel
	glog.Infof("Shutting down..")
}
