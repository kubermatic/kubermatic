package main

import (
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
	kp := kubernetes.NewKubernetesProvider(cps)

	mux.
		Methods("GET").
		Path("/").
		HandlerFunc(handler.StatusOK)

	mux.
		Methods("POST").
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(handler.NewCluster(ctx, kp, cps))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster").
		Handler(handler.Clusters(ctx, kp, cps))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(handler.Cluster(ctx, kp, cps))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}/node").
		Handler(handler.Nodes(ctx, kp, cps))

	http.Handle("/", mux)
	log.Fatal(http.ListenAndServe(":8080", ghandlers.CombinedLoggingHandler(os.Stdout, mux)))
}
