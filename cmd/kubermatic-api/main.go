package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	ghandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/handler"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud"
	"github.com/kubermatic/api/provider/kubernetes"
	"golang.org/x/net/context"
)

func main() {
	ctx := context.Background()
	mux := mux.NewRouter()
	cps := map[string]provider.CloudProvider{
		provider.FakeCloudProvider:         cloud.NewFakeCloudProvider(),
		provider.DigitaloceanCloudProvider: nil,
		// provider.LinodeCloudProvider: nil,
	}
	kps := map[string]provider.KubernetesProvider{
		"fake-1": kubernetes.NewKubernetesFakeProvider("fake-1", cps),
		"fake-2": kubernetes.NewKubernetesFakeProvider("fake-2", cps),
	}

	mux.
		Methods("GET").
		Path("/").
		HandlerFunc(handler.StatusOK)

	mux.
		Methods("POST").
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(handler.NewCluster(ctx, kps, cps))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster").
		Handler(handler.Clusters(ctx, kps, cps))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(handler.Cluster(ctx, kps, cps))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node").
		Handler(handler.Nodes(ctx, kps, cps))

	http.Handle("/", mux)
	log.Fatal(http.ListenAndServe(":8080", ghandlers.CombinedLoggingHandler(os.Stdout, mux)))
}
