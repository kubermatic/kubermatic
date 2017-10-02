package template

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"golang.org/x/crypto/ssh"
)

// FuncMap defines the available functions to kubermatic templates.
var funcs = template.FuncMap{
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
func TxtFuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()

	for name, f := range funcs {
		funcMap[name] = f
	}

	return funcMap
}

// ParseFile creates a new template for the given filenames
// and parses the template definitions from the named files.
func ParseFile(filename string) (*template.Template, error) {
	glog.V(6).Infof("Loading template %q", filename)

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	tpl, err := template.New("base").Funcs(TxtFuncMap()).Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %v", filename, err)
	}

	return tpl, nil
}
