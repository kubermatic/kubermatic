package main

import (
	"io/ioutil"
	"log"
	"os"
)

type Config struct {
	Global GlobalConfig `yaml:"Global"`
	Nodes  []NodeConfig `yaml:"Nodes"`
}

type GlobalConfig struct {
	CA                       Certificate      `yaml:"CA"`
	ServiceAccount           ServiceAccount   `yaml:"ServiceAccount"`
	Kubernetes               KubernetesConfig `yaml:"Kubernetes"`
	SSHKeys                  []string         `yaml:"SSHKeys"`
	APIServerTLSCertificate  Certificate      `yaml:"APIServerTLSCertificate"`
	KubeletClientCertificate Certificate      `yaml:"KubeletClientCertificate"`
}

type Certificate struct {
	Key      string `yaml:"Key"`
	KeyPath  string `yaml:"KeyPath"`
	Cert     string `yaml:"Cert"`
	CertPath string `yaml:"CertPath"`
}

func (c Certificate) GetKey() string {
	if c.Key != "" {
		return c.Key
	}

	b, err := ioutil.ReadFile(c.KeyPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println(err)
		}
	}
	return string(b)
}

func (c Certificate) GetCert() string {
	if c.Cert != "" {
		return c.Cert
	}

	b, err := ioutil.ReadFile(c.CertPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println(err)
		}
		return ""
	}
	return string(b)
}

type ServiceAccount struct {
	Key     string `yaml:"Key"`
	KeyPath string `yaml:"KeyPath"`
}

func (sa ServiceAccount) GetKey() string {
	if sa.Key != "" {
		return sa.Key
	}

	b, err := ioutil.ReadFile(sa.KeyPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println(err)
		}
	}
	return string(b)
}

type KubernetesConfig struct {
	MasterIP                    string `yaml:"MasterIP"`
	Version                     string `yaml:"Version"`
	ServiceNodePortRange        string `yaml:"ServiceNodePortRange"`
	ServiceClusterIPRange       string `yaml:"ServiceClusterIPRange"`
	PodNetworkIPRange           string `yaml:"PodNetworkIPRange"`
	DNSIP                       string `yaml:"DNSIP"`
	DNSDomain                   string `yaml:"DNSDomain"`
	SchedulerKubeconfig         string `json:"SchedulerKubeconfig"`
	ControllerManagerKubeconfig string `json:"ControllerManagerKubeconfig"`
}

type NodeType string

const (
	NodeTypeWorker NodeType = "worker"
	NodeTypeMaster NodeType = "master"
)

type NodeConfig struct {
	Name       string        `yaml:"Name"`
	Type       NodeType      `yaml:"Type"`
	Etcd       EtcdConfig    `yaml:"Etcd"`
	Mounts     []MountConfig `yaml:"Mounts"`
	Network    NetworkConfig `yaml:"Network"`
	Kubeconfig string        `yaml:"Kubeconfig"`
}

type EtcdConfig struct {
	Enabled       bool   `yaml:"Enabled"`
	Version       string `yaml:"Version"`
	DataDirectory string `yaml:"DataDirectory"`
}

type NetworkConfig struct {
	Configure  bool     `yaml:"Configure"`
	Interface  string   `yaml:"Interface"`
	Address    string   `yaml:"Address"`
	Broadcast  string   `yaml:"Broadcast"`
	Gateway    string   `yaml:"Gateway"`
	Domains    []string `yaml:"Domains"`
	DNSServers []string `yaml:"DNSServers"`
	NTPServers []string `yaml:"NTPServers"`
}

type MountConfig struct {
	Name          string `yaml:"Name"`
	Device        string `yaml:"Device"`
	Where         string `yaml:"Where"`
	Type          string `yaml:"Type"`
	DirectoryMode string `yaml:"DirectoryMode"`
}
