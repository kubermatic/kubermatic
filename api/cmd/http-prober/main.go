package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"time"
)

var (
	endpoint  string
	insecure  bool
	retries   int
	retryWait int
	timeout   int
)

const (
	padding = "    "
)

func main() {
	flag.StringVar(&endpoint, "endpoint", "", "The endpoint which should be waited for")
	flag.BoolVar(&insecure, "insecure", false, "Disable certificate validation")
	flag.IntVar(&retries, "retries", 10, "Number of retries")
	flag.IntVar(&retryWait, "retry-wait", 1, "Wait interval in seconds between retries")
	flag.IntVar(&timeout, "timeout", 30, "Timeout in seconds")
	flag.Parse()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(timeout) * time.Second,
	}

	e, err := url.Parse(endpoint)
	if err != nil {
		log.Fatalf("invalid endpoint specified: %v", err)
	}

	req, err := http.NewRequest("GET", e.String(), nil)
	if err != nil {
		log.Fatalf("failed to build request: %v\n", err)
	}

	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			log.Printf("%s resolved to: %s", e.Hostname(), connInfo.Conn.RemoteAddr())
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	for i := 1; i <= retries; i++ {
		if i > 1 {
			time.Sleep(time.Duration(retryWait) * time.Second)
			log.Println()
		}

		log.Printf("[%d/%d] Probing '%s'...\n", i, retries, req.URL.String())
		resp, err := client.Do(req)
		if err != nil {
			log.Printf(padding+"Failed executing request: %v", err)
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			log.Printf(padding+"Failed: '%s' responded with statuscode != 2xx (%d)", req.URL.String(), resp.StatusCode)
			continue
		}

		log.Println("Endpoint is available!")
		os.Exit(0)
	}

	log.Fatalf("Failed: Reached retry limit!\n")
}
