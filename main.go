package main

import (
	"log"
	"net/http"
	"os"

	"golang.org/x/net/context"

	"github.com/gorilla/mux"
	ghandlers "github.com/gorilla/handlers"
	"github.com/kubermatic/api/handler"
)

func main() {
	ctx := context.Background()
	mux := mux.NewRouter()

	mux.
		Methods("GET").
		Path("/").
		HandlerFunc(handler.StatusOK)

	mux.
		Methods("POST").
		Path("/cluster/{provider}").
		Handler(handler.NewCluster(ctx))

	mux.
		Methods("GET").
		Path("/cluster/{provider}").
		Handler(handler.Clusters(ctx))

	http.Handle("/", mux)
	log.Fatal(http.ListenAndServe(":8080", ghandlers.CombinedLoggingHandler(os.Stdout, mux)))
}
