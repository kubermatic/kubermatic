package main

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"path"
	"text/template"

	"github.com/Masterminds/sprig"
	ctconfig "github.com/coreos/container-linux-config-transpiler/config"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

func generateContainerLinuxConfigs(c *cli.Context) error {
	configPath := c.String("config")
	templatePath := c.String("template")
	outputDir := c.String("output-dir")

	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config '%s' : %v", configPath, err)
	}

	templateBytes, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template '%s' : %v", templatePath, err)
	}

	cfg := Config{}
	err = yaml.Unmarshal(configBytes, &cfg)
	if err != nil {
		return err
	}

	certs, err := certutil.ParseCertsPEM([]byte(cfg.Global.CA.GetCert()))
	if err != nil {
		return fmt.Errorf("got an invalid ca cert: %v", err)
	}

	key, err := certutil.ParsePrivateKeyPEM([]byte(cfg.Global.CA.GetKey()))
	if err != nil {
		return fmt.Errorf("got an invalid ca key: %v", err)
	}

	caKp := &triple.KeyPair{
		Cert: certs[0],
		Key:  key.(*rsa.PrivateKey),
	}

	sip, _, err := net.ParseCIDR(cfg.Global.Kubernetes.ServiceClusterIPRange)
	if err != nil {
		return fmt.Errorf("failed to parse service network address '%s': %v", cfg.Global.Kubernetes.ServiceClusterIPRange, err)
	}
	apiserverServiceIP := sip.To4()
	apiserverServiceIP[3]++

	for _, nodeCfg := range cfg.Nodes {
		ip, _, err := net.ParseCIDR(nodeCfg.Network.Address)
		if err != nil {
			return fmt.Errorf("failed to parse network address '%s' for node %s: %v", nodeCfg.Network.Address, nodeCfg.Name, err)
		}

		if nodeCfg.Type == NodeTypeMaster {
			//Generate apiserver tls serving certificates
			if cfg.Global.APIServerTLSCertificate.GetCert() == "" {
				log.Println("generating new apiserver tls certificates")
				apiKp, err := triple.NewServerKeyPair(caKp, nodeCfg.Name, "kubernetes", "default", cfg.Global.Kubernetes.DNSDomain, []string{apiserverServiceIP.String(), nodeCfg.Network.Address, cfg.Global.Kubernetes.MasterIP, ip.String()}, []string{nodeCfg.Name})
				if err != nil {
					return fmt.Errorf("failed to create apiserver tls key pair: %v", err)
				}

				cfg.Global.APIServerTLSCertificate.Cert = string(certutil.EncodeCertPEM(apiKp.Cert))
				cfg.Global.APIServerTLSCertificate.Key = string(certutil.EncodePrivateKeyPEM(apiKp.Key))
			}

			if cfg.Global.KubeletClientCertificate.GetCert() == "" {
				log.Println("generating new apiserver kubelet client certificates")
				kubeletKp, err := triple.NewClientKeyPair(caKp, "apiserver", []string{"system:masters"})
				if err != nil {
					return fmt.Errorf("failed to create kubelet client key pair: %v", err)
				}

				cfg.Global.KubeletClientCertificate.Cert = string(certutil.EncodeCertPEM(kubeletKp.Cert))
				cfg.Global.KubeletClientCertificate.Key = string(certutil.EncodePrivateKeyPEM(kubeletKp.Key))
			}
			if cfg.Global.Kubernetes.ControllerManagerKubeconfig == "" {
				log.Println("creating kubeconfig for controller manager")
				kubeconfig, err := createKubeconfig(fmt.Sprintf("https://%s:6443", cfg.Global.Kubernetes.MasterIP), caKp, "system:kube-controller-manager", []string{"system:kube-controller-manager"})
				if err != nil {
					return err
				}
				cfg.Global.Kubernetes.ControllerManagerKubeconfig = kubeconfig
			}
			if cfg.Global.Kubernetes.SchedulerKubeconfig == "" {
				log.Println("creating kubeconfig for scheduler")
				kubeconfig, err := createKubeconfig(fmt.Sprintf("https://%s:6443", cfg.Global.Kubernetes.MasterIP), caKp, "system:kube-scheduler", []string{"system:kube-scheduler"})
				if err != nil {
					return err
				}
				cfg.Global.Kubernetes.SchedulerKubeconfig = kubeconfig
			}
		}

		if nodeCfg.Kubeconfig == "" {
			log.Println("creating kubeconfig for kubelet")
			kubeconfig, err := createKubeconfig(fmt.Sprintf("https://%s:6443", cfg.Global.Kubernetes.MasterIP), caKp, fmt.Sprintf("system:node:%s", nodeCfg.Name), []string{"system:nodes"})
			if err != nil {
				return err
			}
			nodeCfg.Kubeconfig = kubeconfig
		}

		data := struct {
			Global GlobalConfig
			Node   NodeConfig
			IPV4   string
		}{
			Global: cfg.Global,
			Node:   nodeCfg,
			IPV4:   ip.String(),
		}

		tpl, err := template.New("container-linux").Funcs(sprig.TxtFuncMap()).Parse(string(templateBytes))
		if err != nil {
			return fmt.Errorf("failed to parse %q: %v", templatePath, err)
		}

		buf := bytes.Buffer{}
		if err := tpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to execute template: %v", err)
		}

		cconfig, ast, rep := ctconfig.Parse(buf.Bytes())
		if rep.String() != "" {
			log.Println(rep.String())
		}

		ignConfig, rep := ctconfig.Convert(cconfig, "custom", ast)
		if rep.String() != "" {
			log.Println(rep.String())
		}

		ignBytes, err := json.Marshal(ignConfig)
		if err != nil {
			return err
		}

		ioutil.WriteFile(path.Join(outputDir, "res_"+nodeCfg.Name+".yaml"), buf.Bytes(), 0644)
		ioutil.WriteFile(path.Join(outputDir, "res_ign_"+nodeCfg.Name+".yaml"), ignBytes, 0644)
	}

	log.Println("creating master kubeconfig")
	kubeconfig, err := createKubeconfig(fmt.Sprintf("https://%s:6443", cfg.Global.Kubernetes.MasterIP), caKp, "admin", []string{"system:masters"})
	if err != nil {
		return err
	}
	ioutil.WriteFile(path.Join(outputDir, "res_admin-kubeconfig"), []byte(kubeconfig), 0644)
	return nil
}

func createKubeconfig(address string, ca *triple.KeyPair, commonName string, organizations []string) (string, error) {
	kp, err := triple.NewClientKeyPair(ca, commonName, organizations)
	if err != nil {
		return "", fmt.Errorf("failed to create client certificates for kubeconfig: %v", err)
	}
	kubeconfig := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"default": {
				CertificateAuthorityData: certutil.EncodeCertPEM(ca.Cert),
				Server: address,
			},
		},
		CurrentContext: "default",
		Contexts: map[string]*clientcmdapi.Context{
			"default": {
				Cluster:  "default",
				AuthInfo: "default",
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"default": {
				ClientCertificateData: certutil.EncodeCertPEM(kp.Cert),
				ClientKeyData:         certutil.EncodePrivateKeyPEM(kp.Key),
			},
		},
	}
	kb, err := clientcmd.Write(kubeconfig)
	if err != nil {
		return "", err
	}
	return string(kb), nil
}
