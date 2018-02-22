package kubernetes

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	texttemplate "text/template"

	"github.com/Masterminds/sprig"
	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"golang.org/x/crypto/ssh"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// Template holds the actual cluster template
type Template struct {
	tpl *texttemplate.Template
}

// ParseFile creates a new template for the given filenames
// and parses the template definitions from the named files.
func ParseFile(filename string) (*Template, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	tpl, err := texttemplate.New("base").Funcs(TxtFuncMap()).Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %v", filename, err)
	}

	return &Template{tpl}, nil
}

// Execute applies a parsed template to the specified data object,
// and stores the result in the value pointed to by v.
func (t *Template) Execute(data interface{}, object v1.Object) (string, error) {
	var buf bytes.Buffer
	if err := t.tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed executing template: %v", err)
	}
	b := buf.Bytes()

	jsonBytes, err := yaml.ToJSON(b)
	if err != nil {
		return "", fmt.Errorf("failed converting yaml to json: %v", err)
	}

	if err := json.Unmarshal(jsonBytes, object); err != nil {
		return "", fmt.Errorf("failed unmarshaling: %v", err)
	}

	if object.GetLabels() == nil {
		object.SetLabels(map[string]string{})
	}
	if object.GetAnnotations() == nil {
		object.SetAnnotations(map[string]string{})
	}

	return string(jsonBytes), nil
}

// FuncMap defines the available functions to kubermatic templates.
var funcs = texttemplate.FuncMap{
	"apiBytesToB64":      apiBytesToB64,
	"apiBytesToString":   apiBytesToString,
	"pemEncodePublicKey": pemEncodePublicKey,
}

func apiBytesToB64(b kubermaticv1.Bytes) string {
	return b.Base64()
}

func pemEncodePublicKey(b kubermaticv1.Bytes) string {
	k, _, _, _, err := ssh.ParseAuthorizedKey(b)
	if err != nil {
		glog.Errorf("Failed to parse authorized key: %v", err)
		return ""
	}
	ck := k.(ssh.CryptoPublicKey)
	pk := ck.CryptoPublicKey()
	rsakey, ok := pk.(*rsa.PublicKey)
	if !ok {
		glog.Errorf("key is not of type rsa.PublicKey")
		return ""
	}

	publicBytes, err := x509.MarshalPKIXPublicKey(rsakey)
	if err != nil {
		glog.Errorf("failed to marshal public key: %v", err)
		return ""
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Bytes: publicBytes,
		Type:  "PUBLIC KEY",
	})

	return string(pemBytes)
}

func apiBytesToString(b kubermaticv1.Bytes) string {
	return string(b)
}

// TxtFuncMap returns an aggregated template function map. Currently (custom functions + sprig)
func TxtFuncMap() texttemplate.FuncMap {
	funcMap := sprig.TxtFuncMap()

	for name, f := range funcs {
		funcMap[name] = f
	}

	return funcMap
}
