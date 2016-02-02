package main

import (
	"log"
	"net/http"
	"os"

	ghandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/handler"
	"github.com/kubermatic/api/provider/cluster"
	"golang.org/x/net/context"
)

func main() {
	ctx := context.Background()
	mux := mux.NewRouter()
	cp := cluster.NewClusterProvider()

	mux.
		Methods("GET").
		Path("/").
		HandlerFunc(handler.StatusOK)

	mux.
		Methods("POST").
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(handler.NewCluster(ctx, cp))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/").
		Handler(handler.Clusters(ctx, cp))

	mux.
		Methods("GET").
		Path("/api/v1/dc/{dc}/cluster/{cluster}").
		Handler(handler.Cluster(ctx, cp))

	http.Handle("/", mux)
	log.Fatal(http.ListenAndServe(":8080", ghandlers.CombinedLoggingHandler(os.Stdout, mux)))
}
