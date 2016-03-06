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
var FuncMap template.FuncMap = template.FuncMap{
	"readLinesTemplate":  readLinesTemplate,
	"readBase64":         readBase64,
	"readBase64Gzip":     readBase64Gzip,
	"readBase64Template": readBase64Template,
}

// Data is the struct defining kubermatic template variables.
type Data struct {
	SSHAuthorizedKeys []string
	EtcdURL           string
	APIServerURL      string
	Region            string
	KubeletToken      string
	Name              string
}

func readLinesTemplate(data interface{}, path string) (lines []string) {
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
		t.Execute(&buf, data)

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

		io.Copy(b64, f)
		f.Close()
	}

	b64.Close()
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

		io.Copy(gz, f)
		f.Close()
	}

	gz.Close()
	b64.Close()

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
	b64.Close()

	return buf.String()
}
