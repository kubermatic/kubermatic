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
	mux.Handle("/cluster", handler.NewCluster(ctx)).Methods("POST")
	mux.Handle("/cluster/{provider}", handler.Clusters(ctx)).Methods("GET")
	http.Handle("/", mux)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
