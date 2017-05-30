package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

func env(name, def string) (val string) {
	val = os.Getenv(name)
	if val == "" {
		val = def
	}
	return val
}

func mustEnv(name string) (val string) {
	val = os.Getenv(name)
	if val == "" {
		panic(fmt.Sprintf("Environment variable %s must be set", name))
	}
	return
}

func kubectlCreate(filename string, data interface{}) error {
	tpl, err := template.ParseFiles(filename)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return err
	}

	kubectl := exec.Command("kubectl", "create", "-f", "-")

	stdin, err := kubectl.StdinPipe()
	if err != nil {
		return err
	}
	kubectl.Stdout = os.Stdout
	kubectl.Stderr = os.Stderr

	if err := kubectl.Start(); err != nil {
		return err
	}

	if _, err := io.Copy(stdin, &buf); err != nil {
		log.Println(err)
	}

	err = stdin.Close()
	if err != nil {
		return err
	}

	return kubectl.Wait()
}

func deployKubeSystem() error {
	return kubectlCreate("namespace/kube-system-ns.yaml", struct{}{})
}

func deployDNS() error {
	data := struct {
		Replicas  string
		DNSDomain string
		MasterURL string
		ClusterIP string
	}{
		Replicas:  env("REPLICAS", "1"),
		DNSDomain: env("DNS_DOMAIN", "cluster.local"),
		MasterURL: os.Getenv("MASTER_URL"),
		ClusterIP: env("CLUSTER_IP", "10.10.10.10"),
	}

	if err := kubectlCreate("namespace/kube-system-ns.yaml", data); err != nil {
		log.Println(err)
	}

	if err := kubectlCreate("dns/skydns-rc.yaml", data); err != nil {
		return err
	}

	if err := kubectlCreate("dns/skydns-svc.yaml", data); err != nil {
		log.Println(err)
	}

	return nil
}

func deployIngress() error {
	data := struct {
		Region         string
		NodeFloatingIP string
	}{
		Region: mustEnv("REGION"),
	}

	o, err := exec.Command("kubectl", "config", "view", "-o=jsonpath={.current-context}").Output()
	if err != nil {
		return err
	}
	currentContext := string(o)
	dc := "do-" + data.Region
	if dc != currentContext {
		return fmt.Errorf("REGION value %q does not match current kubectl context %q", dc, currentContext)
	}

	domain := fmt.Sprintf("ingress.do-%s.kubermatic.io", data.Region)
	println(fmt.Sprintf("Trying to resolve %s", domain))
	ips, err := net.LookupHost(domain)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return fmt.Errorf("No IPs for %s", domain)
	}
	println(fmt.Sprintf("Using first ip of %v", ips))
	data.NodeFloatingIP = ips[0]

	files := []string{
		"ingress-namespace.yaml",
		"default-backend-service.yaml", "default-backend-rc.yaml",
		"dhparam-secret.yaml", fmt.Sprintf("do-%s-kubermatic-certs-secret.yaml", data.Region),
		"do-token-secret.yaml",
		"controller-rc.yaml",
	}

	print("Trying to delete namespace ingress: ")
	cmd := exec.Command("kubectl", "delete", "namespace", "ingress")
	o, err = cmd.CombinedOutput()
	if err != nil {
		return err
	}
	print(strings.TrimSuffix(string(o), "\n"))

	for {
		get := exec.Command("kubectl", "get", "namespace", "ingress")
		err = get.Run()
		if err != nil {
			return err
		}
		if !get.ProcessState.Success() {
			break
		}
		print(".")
		time.Sleep(1 * time.Second)
	}
	println()

	for _, f := range files {
		print(fmt.Sprintf("Trying to create ingress/%s: ", f))
		if err := kubectlCreate("ingress/"+f, data); err != nil {
			return err
		}
	}

	return nil
}

func usage(cmds []cmd) {
	println("Available cmds:")
	for _, c := range cmds {
		println("- " + c.name)
	}
}

type cmd struct {
	name string
	run  func() error
}

func main() {
	cmds := []cmd{
		{"kube-system", deployKubeSystem},
		{"dns", deployDNS},
		{"ingress", deployIngress},
	}
	if len(os.Args) == 1 {
		usage(cmds)
		os.Exit(0)
	}

	n := os.Args[1]
	for _, c := range cmds {
		if n != c.name && n != "all" {
			continue
		}

		println(fmt.Sprintf("Running action %s", c.name))
		err := c.run()
		if err != nil {
			log.Fatal(err)
		}

		if n != "all" {
			return
		}
	}

	if n != "all" {
		log.Fatalf("Action %s not found", n)
		usage(cmds)
		os.Exit(1)
	}
}
