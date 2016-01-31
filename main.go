package main

import (
	"log"
	"net/http"

	"golang.org/x/net/context"

	"github.com/gorilla/mux"
	"github.com/sttts/kubermatic-api/handler"
)

func main() {
	ctx := context.Background()
	mux := mux.NewRouter()

	mux.
		Methods("POST").
		Path("/cluster/{provider}").
		Handler(handler.NewCluster(ctx))

	mux.
		Methods("GET").
		Path("/cluster/{provider}").
		Handler(handler.Clusters(ctx))

	http.Handle("/", mux)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
