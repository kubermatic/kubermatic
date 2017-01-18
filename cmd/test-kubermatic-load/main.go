package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kubermatic/api"
)

var (
	jwtFlag             = flag.String("jwt-auth", "", "The String of the authorization header")
	maxNodesFlag        = flag.Int("node-count", 0, "The amount of nodes to create in one cluster (node-count*cluster-count)")
	maxClustersFlag     = flag.Int("cluster-count", 0, "The amount of clusters to deploy")
	dcFlag              = flag.String("datacenter-name", "master", "The master dc")
	maxAsyncFlag        = flag.Int("max-workers", 10, "The amount of maximum concurrent requests")
	retryNSIntervalFlag = flag.Int64("ns-retry-interval", 10, "The duration in seconds to wait between namespace alive requests")
	domainFlag          = flag.String("domain", "dev.kubermatic.io", "The domain to api is running on")
)

// setAuth sets the jwt token int a requests Authorization header field
func setAuth(r *http.Request) {
	r.Header.Add("Authorization", *jwtFlag)
}

// createNodes creates nodes
func createNodes(nodeCount int, cluster api.Cluster, client *http.Client) error {
	// Create no nodes
	if nodeCount < 1 {
		return nil
	}

	// Create node request
	req, err := http.NewRequest("POST", fmt.Sprintf("https://"+*domainFlag+"/api/v1/dc/"+*dcFlag+"/cluster/%s/node", cluster.Metadata.Name),
		// The sshKeys are fix.
		strings.NewReader(fmt.Sprintf(`{"instances":%d,"spec":{"digitalocean":{"sshKeys":["80:ba:7a:3b:3f:89:b1:b4:cd:b8:b4:fb:6c:a4:62:d0"],"size":"512mb"},"dc":"do-ams2"}}`, nodeCount)))
	if err != nil {
		return err
	}
	setAuth(req)

	_, err = client.Do(req)

	return err
}

// createProvider creates a new Cloud Provider for a cluster.
// This should only be called after the NS is created.
func createProvider(cluster api.Cluster, client *http.Client) error {
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://"+*domainFlag+"/api/v1/dc/"+*dcFlag+"/cluster/%s/cloud", cluster.Metadata.Name),
		strings.NewReader(`{"dc":"do-ams2","digitalocean":{"sshKeys":["80:ba:7a:3b:3f:89:b1:b4:cd:b8:b4:fb:6c:a4:62:d0"],"token":"0f76d511c5f5c8730b18d588a07cd56aa78fc8a6ddabbc168eceaaa9c7a12892"}}`))
	if err != nil {
		return err
	}
	setAuth(req)

	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}

// waitNS waits for the Namespace to get created.
func waitNS(cl api.Cluster, client *http.Client) error {
	for {
		req, err := http.NewRequest("GET", "https://"+*domainFlag+"/api/v1/dc/"+*dcFlag+"/cluster/"+cl.Metadata.Name, nil)
		if err != nil {
			return err
		}
		setAuth(req)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		// Read all Data from the body,
		// We cloud also use a tee reader but this would have been a bit overkill.
		data, _ := ioutil.ReadAll(resp.Body)

		// We use the cluster state to get notified when a new NS gets created.
		// When a NS gets created the cluster will revice an cluster URL.
		var clusterState api.Cluster
		if err = json.NewDecoder(bytes.NewReader(data)).Decode(&clusterState); err != nil {
			log.Println(string(data))
			return err
		}

		// Check if cluster NS is created and running
		if clusterState.Address.URL != "" && clusterState.Status.Phase == api.RunningClusterStatusPhase {
			break
		}
		log.Println("Waiting for NS to get created ....")
		// Sleep until next check
		time.Sleep(time.Second * time.Duration(*retryNSIntervalFlag))
	}
	return nil
}

// deleteCluster deletes all clusters for a given user
func deleteCluster(cluster api.Cluster, client *http.Client) error {
	log.Printf("Deleting %q\n", cluster.Metadata.Name)
	req, err := http.NewRequest("DELETE", "https://"+*domainFlag+"/api/v1/dc/"+*dcFlag+"/cluster/"+cluster.Metadata.Name, nil)
	if err != nil {
		return err
	}
	setAuth(req)
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	return nil
}

// up is the main entry point for the up method
func up(maxClusters, maxNodes int) error {
	client := &http.Client{}

	log.Printf("Creating %d clusters with %d nodes inside in total %d nodes",
		maxClusters, maxNodes, maxNodes*maxClusters)

	// Sync after all workers are done
	waitAll := sync.WaitGroup{}
	waitAll.Add(maxClusters)

	// Orchestrate all workers.
	// All go routines are started simultaniously.
	// An active worker puts one "ticket" in the channel.
	// When the worker is done the ticket is removed and other workers starts.
	// Due to the fact that a channel blocks when
	//  it's full only a specific amout of workers are running actively at the
	//  same time (maxAsyncFlag)
	done := make(chan struct{}, *maxAsyncFlag)
	// Start all workers
	for i := 0; i < maxClusters; i++ {
		log.Printf("started worker-%d", i)
		// Start go routine, reevaluate x every time
		go func(x int) {
			// inner is used to quickly execute teardown behaviour
			// It is not implemented as a function due to its heavy scope
			inner := func() {
				// Place ticket
				done <- struct{}{}
				log.Printf("request-%d", x)

				// Create cluster request the clustername will be "test-{number}"
				req, err := http.NewRequest("POST", "https://"+*domainFlag+"/api/v1/dc/"+*dcFlag+"/cluster",
					strings.NewReader(fmt.Sprintf(`{"spec":{"humanReadableName":"test-%d"}}`, x)))
				if err != nil {
					<-done
					return
				}
				setAuth(req)

				resp, err := client.Do(req)
				// Remove ticket
				time.Sleep(40 * time.Second)
				<-done
				if err != nil {
					log.Println(err)
					return
				}

				// Read body, tee reader is overkill
				data, _ := ioutil.ReadAll(resp.Body)
				var cluster api.Cluster
				if err = json.NewDecoder(bytes.NewReader(data)).Decode(&cluster); err != nil {
					log.Println(string(data))
					log.Println(err)
					return
				}

				log.Printf("Created Cluster \"test-%d\"\n", x)

				// wait for NS to not get errors when setting the cloud provider
				if err = waitNS(cluster, client); err != nil {
					log.Println(err)
					return
				}

				// TODO(realfake): Move ticket removal here?

				if err = createProvider(cluster, client); err != nil {
					log.Println(err)
					return
				}

				if err = createNodes(maxNodes, cluster, client); err != nil {
					log.Println(err)
					return
				}
			}

			// Execute inner and sync
			inner()
			waitAll.Done()
		}(i)
	}

	// Wait for all workers to finish
	waitAll.Wait()
	return nil
}

// purge is the main entry point for the merge command
func purge() error {
	client := &http.Client{}

	// Get clusters list
	req, err := http.NewRequest("GET", "https://"+*domainFlag+"/api/v1/dc/"+*dcFlag+"/cluster", nil)
	if err != nil {
		return err
	}

	setAuth(req)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	var clusters []api.Cluster
	if err = json.NewDecoder(resp.Body).Decode(&clusters); err != nil {
		return err
	}

	// Same pattern as in up
	done := make(chan struct{}, *maxAsyncFlag)
	waitAll := sync.WaitGroup{}
	waitAll.Add(len(clusters))
	for _, cluster := range clusters {
		go func(cl api.Cluster) {
			// Place ticket
			done <- struct{}{}
			log.Println(deleteCluster(cl, client))
			// Remove ticket
			<-done
			waitAll.Done()
		}(cluster)
	}
	// Wait for all workers
	waitAll.Wait()
	return nil
}

func main() {
	flag.Parse()
	printError := func() {
		log.Printf("Wrong usage. Use:\n\n\t %s [up|purge]\n\n", os.Args[0])
		os.Exit(1)
	}

	if len(flag.Args()) < 1 {
		printError()
	}

	if *jwtFlag == "" {
		log.Println("Please specify a jwt flag")
		os.Exit(1)
	}

	var err error
	switch flag.Arg(0) {
	case "up":
		err = up(*maxClustersFlag, *maxNodesFlag)
	case "purge":
		err = purge()
	default:
		printError()
	}

	if err != nil {
		log.Fatal(err)
	}
}
