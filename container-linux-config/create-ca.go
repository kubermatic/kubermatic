package main

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/urfave/cli"
	certutil "k8s.io/client-go/util/cert"
	_ "k8s.io/client-go/util/cert/triple"
)

func createCA(c *cli.Context) error {
	commonName := c.String("common-name")
	outputDir := c.String("output-dir")

	key, err := certutil.NewPrivateKey()
	if err != nil {
		return fmt.Errorf("unable to create a private key for a new CA: %v", err)
	}

	config := certutil.Config{CommonName: commonName}

	caCert, err := certutil.NewSelfSignedCACert(config, key)
	if err != nil {
		return fmt.Errorf("unable to create a self-signed certificate for a new CA: %v", err)
	}

	certPath := path.Join(outputDir, "ca.crt")
	if err := ioutil.WriteFile(certPath, certutil.EncodeCertPEM(caCert), 0644); err != nil {
		return fmt.Errorf("failed to write '%s': %v", certPath, err)
	}
	keyPath := path.Join(outputDir, "ca.key")
	if err := ioutil.WriteFile(keyPath, certutil.EncodePrivateKeyPEM(key), 0644); err != nil {
		return fmt.Errorf("failed to write '%s': %v", keyPath, err)
	}

	return nil
}
