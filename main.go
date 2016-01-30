package main

import (
	"log"
	"net/http"

	"golang.org/x/net/context"

	"github.com/sttts/kubermatik-api/handler"
)

func main() {
	ctx := context.Background()
	http.Handle("/newCluster", handler.NewClusterHandler(ctx))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
