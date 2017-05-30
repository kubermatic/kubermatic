package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"text/template"
)

var funcMap = template.FuncMap{
	"readLines":          readLines,
	"readBase64":         readBase64,
	"readBase64Gzip":     readBase64Gzip,
	"readBase64Template": readBase64Template,
}

var data struct {
	Name            string
	DiscoveryURL    string
	K8sDiscoveryURL string
	MasterHost      string
	AdminToken      string
	KubeletToken    string
	Region          string
	FloatingIP      string
	APIToken        string
}

func main() {
	dumpOnly := flag.Bool("dump-only", false, "Only dump cloud config, don't do anything.")
	flag.Parse()

	data.Name = os.Getenv("NAME")
	data.DiscoveryURL = os.Getenv("DISCOVERY_URL")
	data.K8sDiscoveryURL = os.Getenv("K8S_DISCOVERY_URL")
	data.MasterHost = fmt.Sprintf("do-%s.kubermatic.io", os.Getenv("REGION"))
	data.KubeletToken = os.Getenv("KUBELET_TOKEN")
	data.AdminToken = os.Getenv("ADMIN_TOKEN")
	data.Region = os.Getenv("REGION")
	data.APIToken = os.Getenv("TOKEN")

	ips, err := net.LookupIP(data.MasterHost)
	if err != nil {
		panic(err)
	}
	if len(ips) != 1 {
		panic(fmt.Errorf("Master host %s must resolve to exactly one ip, not: %v", data.MasterHost, ips))
	}
	data.FloatingIP = ips[0].String()

	switch {
	case data.DiscoveryURL == "":
		panic(errors.New("DISCOVERY_URL is undefined"))
	case data.K8sDiscoveryURL == "":
		panic(errors.New("K8S_DISCOVERY_URL is undefined"))
	case data.KubeletToken == "":
		panic(errors.New("KUBELET_TOKEN is undefined"))
	case data.AdminToken == "":
		panic(errors.New("ADMIN_TOKEN is undefined"))
	}

	tpl, err := readTemplate(flag.Arg(0))
	if err != nil {
		panic(err)
	}

	if *dumpOnly {
		fmt.Printf("%s\n", tpl)
		os.Exit(0)
	}

	json := fmt.Sprintf(`{
  "region": "%s",
  "image": "coreos-stable",
  "size": "%s",
  "private_networking": "true",
  "ssh_keys": %s,
  "name": "%s",
  "user_data": "%s"
}`,
		os.Getenv("REGION"),
		os.Getenv("SIZE"),
		jsonSlice(strings.Split(os.Getenv("SSH_FINGERPRINT"), `,`)),
		os.Getenv("NAME"),
		escape(tpl),
	)

	req, err := http.NewRequest("POST", "https://api.digitalocean.com/v2/droplets", strings.NewReader(json))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("TOKEN")))

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer func() { err := res.Body.Close(); _ = err }()

	fmt.Fprintf(os.Stderr, "Response Status: %v\n", res.Status)
	fmt.Fprintf(os.Stderr, "Response Headers: %+v\n", res.Header)
	_, err = io.Copy(os.Stdout, res.Body)
	if err != nil {
		panic(err)
	}
}

func readTemplate(filename string) (string, error) {
	t, err := template.
		New(filename).
		Funcs(funcMap).
		ParseFiles(filename)

	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func readBase64Template(filename string) string {
	t, err := template.ParseFiles(filename)

	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	b64 := base64.NewEncoder(base64.StdEncoding, &buf)
	if err := t.Execute(b64, data); err != nil {
		panic(err)
	}

	defer func() { err := b64.Close(); _ = err }()

	return buf.String()
}

func readLines(path string) (lines []string) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := scanner.Text()
		t, err := template.New("line").Parse(text)
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer
		err = t.Execute(&buf, data)
		if err != nil {
			log.Panic(err)
		}

		lines = append(lines, buf.String())
	}

	return
}

func escape(line string) string {
	line = strings.Replace(line, `\`, `\\`, -1)
	line = strings.Replace(line, `"`, `\"`, -1)
	return line
}

func readBase64(paths ...string) string {
	var buf bytes.Buffer
	b64 := base64.NewEncoder(base64.StdEncoding, &buf)

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			panic(err)
		}

		_, err = io.Copy(b64, f)
		if err != nil {
			log.Panic(err)
		}
		err = f.Close()
		if err != nil {
			log.Panic(err)
		}
	}

	defer func() { err := b64.Close(); _ = err }()
	return buf.String()
}

func readBase64Gzip(paths ...string) string {
	var buf bytes.Buffer
	b64 := base64.NewEncoder(base64.StdEncoding, &buf)
	gz := gzip.NewWriter(b64)

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			panic(err)
		}

		_, err = io.Copy(gz, f)
		if err != nil {
			log.Panic(err)
		}
		err = f.Close()
		if err != nil {
			log.Panic(err)
		}
	}

	defer func() { err := gz.Close(); _ = err }()
	defer func() { err := b64.Close(); _ = err }()

	return buf.String()
}

func jsonSlice(ss []string) string {
	for i, s := range ss {
		ss[i] = `"`
		ss[i] += s
		ss[i] += `"`
	}
	return `[` + strings.Join(ss, `,`) + `]`
}
