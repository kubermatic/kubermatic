package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"os/exec"
	"syscall"
	"time"

	httpproberapi "github.com/kubermatic/kubermatic/api/cmd/http-prober/api"
)

const (
	padding = "    "
)

func main() {
	var (
		endpoint   string
		insecure   bool
		retries    int
		retryWait  int
		timeout    int
		commandRaw string
	)
	flag.StringVar(&endpoint, "endpoint", "", "The endpoint which should be waited for")
	flag.BoolVar(&insecure, "insecure", false, "Disable certificate validation")
	flag.IntVar(&retries, "retries", 10, "Number of retries")
	flag.IntVar(&retryWait, "retry-wait", 1, "Wait interval in seconds between retries")
	flag.IntVar(&timeout, "timeout", 30, "Timeout in seconds")
	flag.StringVar(&commandRaw, "command", "", "If passed, the http prober will exec this command. Must be json encoded")
	flag.Parse()

	var command *httpproberapi.Command
	if commandRaw != "" {
		command = &httpproberapi.Command{}
		if err := json.Unmarshal([]byte(commandRaw), command); err != nil {
			log.Fatalf("failed to deserialize command: %v", err)
		}
	}

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

		log.Printf("[%02d/%02d] Probing '%s'...\n", i, retries, req.URL.String())
		resp, err := client.Do(req)
		if err != nil {
			log.Printf(padding+"Failed executing request: %v", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			log.Printf(padding+"Failed: '%s' responded with statuscode != 2xx (%d)", req.URL.String(), resp.StatusCode)
			continue
		}

		log.Println("Endpoint is available!")
		if command != nil {
			commandFullPath, err := exec.LookPath(command.Command)
			if err != nil {
				log.Fatalf("failed to look up full path for command %q: %v", command.Command, err)
			}
			// First arg should be the filename of the command being executed, quote from execve(2):
			// `By convention, the first of these strings (i.e., argv[0]) should contain the filename associated with the file being executed`
			args := append([]string{command.Command}, command.Args...)
			if err := syscall.Exec(commandFullPath, args, os.Environ()); err != nil {
				log.Fatalf("failed to execute command: %v", err)
			}
		}
		os.Exit(0)
	}

	log.Fatalf("Failed: Reached retry limit!\n")
}
