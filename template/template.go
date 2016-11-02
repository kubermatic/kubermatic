package template

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"os"
	"text/template"
)

// FuncMap defines the available functions to kubermatic templates.
var FuncMap = template.FuncMap{
	"readLinesTemplate":  readLinesTemplate,
	"readBase64":         readBase64,
	"readBase64Gzip":     readBase64Gzip,
	"readBase64Template": readBase64Template,
}

// Data is the struct defining kubermatic template variables.
type Data struct {
	DC                string
	ClusterName       string
	SSHAuthorizedKeys []string
	EtcdURL           string
	APIServerURL      string
	Region            string
	Name              string
	ClientKey         string
	ClientCert        string
	RootCACert        string
	ApiserverPubSSH   string
	ApiserverToken    string
	FlannelCIDR       string
}

func readLinesTemplate(data interface{}, path string) (lines []string) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		text := scanner.Text()
		t, err := template.New("line").Parse(text)
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			panic(err)
		}

		lines = append(lines, buf.String())
	}

	return
}

func readBase64(paths ...string) string {
	var buf bytes.Buffer
	b64 := base64.NewEncoder(base64.StdEncoding, &buf)

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(b64, f); err != nil {
			panic(err)
		}

		if err := f.Close(); err != nil {
			panic(err)
		}
	}

	if err := b64.Close(); err != nil {
		panic(err)
	}

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

		if _, err := io.Copy(gz, f); err != nil {
			panic(err)
		}

		if err := f.Close(); err != nil {
			panic(err)
		}
	}

	if err := gz.Close(); err != nil {
		panic(err)
	}

	if err := b64.Close(); err != nil {
		panic(err)
	}

	return buf.String()
}

func readBase64Template(data interface{}, filename string) string {
	t, err := template.ParseFiles(filename)

	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	b64 := base64.NewEncoder(base64.StdEncoding, &buf)
	if err := t.Execute(b64, data); err != nil {
		panic(err)
	}

	if err := b64.Close(); err != nil {
		panic(err)
	}

	return buf.String()
}
