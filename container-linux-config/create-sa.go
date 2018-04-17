package main

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/urfave/cli"
	_ "k8s.io/client-go/util/cert/triple"
)

func createSA(c *cli.Context) error {
	outputDir := c.String("output-dir")

	priv, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		return err
	}

	saKey := x509.MarshalPKCS1PrivateKey(priv)
	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: saKey,
	}

	saPath := path.Join(outputDir, "sa.key")
	if err := ioutil.WriteFile(saPath, pem.EncodeToMemory(&block), 0644); err != nil {
		return fmt.Errorf("failed to write '%s': %v", saPath, err)
	}

	return nil
}
