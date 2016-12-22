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
	jwtFlag         = flag.String("jwt", "", "The String of the Authorization: header")
	maxNodesFlag    = flag.Int("nodes", 0, "Spcifies the amount of nodes to create in one cluster (nodes*clusters)")
	maxClustersFlag = flag.Int("clusters", 0, "Spcifies the amount of clusters to deploy")
	dcFlag          = flag.String("datacenter", "master", "use this to specify a datacenter")
	maxAsyncFlag    = flag.Int("max-async", 10, "Spcifies the amount of request running at the same time")
)

func setAuth(r *http.Request) {
	r.Header.Add("Authorization", *jwtFlag)
}

func createNodes(nodeCount int, cluster api.Cluster, client *http.Client) error {
	if nodeCount < 1 {
		return nil
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("https://dev.kubermatic.io/api/v1/dc/"+*dcFlag+"/cluster/%s/node", cluster.Metadata.Name),
		strings.NewReader(fmt.Sprintf(`{"instances":%d,"spec":{"digitalocean":{"sshKeys":["80:ba:7a:3b:3f:89:b1:b4:cd:b8:b4:fb:6c:a4:62:d0"],"size":"512mb"},"dc":"do-ams2"}}`, nodeCount)))
	if err != nil {
		return err
	}

	setAuth(req)

	_, err = client.Do(req)

	return err
}

func createProvider(cluster api.Cluster, client *http.Client) error {
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://dev.kubermatic.io/api/v1/dc/"+*dcFlag+"/cluster/%s/cloud", cluster.Metadata.Name),
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

func waitNS(id int, cl api.Cluster, client *http.Client) error {
	for {
		req, err := http.NewRequest("GET", "https://dev.kubermatic.io/api/v1/dc/"+*dcFlag+"/cluster/"+cl.Metadata.Name, nil)
		if err != nil {
			return err
		}
		setAuth(req)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		data, _ := ioutil.ReadAll(resp.Body)
		var clusterState api.Cluster
		if err = json.NewDecoder(bytes.NewReader(data)).Decode(&clusterState); err != nil {
			log.Println(string(data))
			return err
		}
		if clusterState.Address.URL != "" && clusterState.Status.Phase == api.RunningClusterStatusPhase {
			break
		}
		log.Println("Waiting for NS to get created ....")
		time.Sleep(time.Second * 10)
	}
	return nil
}

func deleteCluster(cluster api.Cluster, client *http.Client) error {
	log.Printf("Deleting %q\n", cluster.Metadata.Name)
	req, err := http.NewRequest("DELETE", "https://dev.kubermatic.io/api/v1/dc/"+*dcFlag+"/cluster/"+cluster.Metadata.Name, nil)
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

func up(maxClusters, maxNodes int) error {
	client := &http.Client{}

	log.Printf("Creating %d clusters with %d nodes inside in total %d nodes",
		maxClusters, maxNodes, maxNodes*maxClusters)

	waitAll := sync.WaitGroup{}
	waitAll.Add(maxClusters)
	done := make(chan struct{}, *maxAsyncFlag)
	for i := 0; i < maxClusters; i++ {
		log.Printf("started worker-%d", i)
		go func(x int) {
			inner := func() {
				done <- struct{}{}
				log.Printf("request-%d", x)
				req, err := http.NewRequest("POST", "https://dev.kubermatic.io/api/v1/dc/"+*dcFlag+"/cluster",
					strings.NewReader(fmt.Sprintf(`{"spec":{"humanReadableName":"test-%d"}}`, x)))
				if err != nil {
					<-done
					return
				}
				setAuth(req)
				resp, err := client.Do(req)
				if err != nil {
					log.Println(err)
					<-done
					return
				}

				data, _ := ioutil.ReadAll(resp.Body)
				var cluster api.Cluster
				if err = json.NewDecoder(bytes.NewReader(data)).Decode(&cluster); err != nil {
					log.Println(string(data))
					log.Println(err)
					<-done
					return
				}
				<-done
				log.Printf("Created Cluster \"test-%d\"\n", x)

				if err = waitNS(i, cluster, client); err != nil {
					log.Println(err)
					return
				}

				if err = createProvider(cluster, client); err != nil {
					log.Println(err)
					return
				}
				if err = createNodes(maxNodes, cluster, client); err != nil {
					log.Println(err)
					return
				}
			}
			inner()
			waitAll.Done()
		}(i)
	}
	waitAll.Wait()
	return nil
}

func purge() error {
	client := &http.Client{}

	// Get clusters list
	req, err := http.NewRequest("GET", "https://dev.kubermatic.io/api/v1/dc/"+*dcFlag+"/cluster", nil)
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

	done := make(chan struct{}, *maxAsyncFlag)
	for _, cluster := range clusters {
		func(cl api.Cluster) {
			done <- struct{}{}
			log.Println(deleteCluster(cluster, client))
			<-done
		}(cluster)
	}
	return nil
}

func main() {
	flag.Parse()
	printError := func() {
		log.Printf("Wrong usage usage.\n\n\t %s [up|purge]\n\n", os.Args[0])
		os.Exit(1)
	}

	if len(flag.Args()) < 1 {
		printError()
	}

	if *jwtFlag == "" {
		log.Printf("Please specify a jwt flag")
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
