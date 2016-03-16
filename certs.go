package api

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/cloudflare/cfssl/signer"
	"github.com/cloudflare/cfssl/signer/local"
)

// CreateKeyCert creates a private/publich RSA key pair with 2048 bits and the given CN.
func (c *Cluster) CreateKeyCert(cn string) (*KeyCert, error) {
	// create key and csr
	req := csr.CertificateRequest{
		CN: cn,
		KeyRequest: &csr.BasicKeyRequest{
			A: "rsa",
			S: 2048,
		},
	}
	gen := csr.Generator{
		Validator: func(req *csr.CertificateRequest) error {
			return nil
		},
	}
	keyCSR, key, err := gen.ProcessRequest(&req)
	if err != nil {
		return nil, fmt.Errorf("error creating %q key and csr: %v", cn, err)
	}

	// sign key with root CA
	caKey, err := helpers.ParsePrivateKeyPEM(c.Status.RootCA.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private ca key: %v", err)
	}
	caCert, err := helpers.ParseCertificatePEM(c.Status.RootCA.Cert)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ca cert: %v", err)
	}
	policy := config.Signing{
		Profiles: map[string]*config.SigningProfile{},
		Default:  config.DefaultConfig(),
	}
	policy.Default.ExpiryString = fmt.Sprintf("%dh", 24*365*10)
	s, err := local.NewSigner(caKey, caCert, signer.DefaultSigAlgo(caKey), &policy)
	if err != nil {
		return nil, fmt.Errorf("error creating signer: %v", err)
	}
	cert, err := s.Sign(signer.SignRequest{
		Request: string(keyCSR),
	})
	if err != nil {
		return nil, fmt.Errorf("error signing %q crt: %v", cn, err)
	}

	return &KeyCert{key, cert}, nil
}

// MarshalJSON adds base64 json encoding to the Bytes type.
func (bs Bytes) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", base64.StdEncoding.EncodeToString(bs))), nil
}

// UnmarshalJSON adds base64 json decoding to the Bytes type.
func (bs *Bytes) UnmarshalJSON(src []byte) error {
	if len(src) < 2 {
		return errors.New("base64 string expected")
	}
	if src[0] != '"' || src[len(src)-1] != '"' {
		return errors.New("\" quotations expected")
	}
	if len(src) == 2 {
		*bs = nil
		return nil
	}
	var err error
	*bs, err = base64.StdEncoding.DecodeString(string(src[1 : len(src)-1]))
	if err != nil {
		return err
	}
	return nil
}

// Base64 converts a Bytes instance to a base64 string.
func (bs Bytes) Base64() string {
	if []byte(bs) == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(bs))
}

// NewBytes creates a Bytes instance from a base64 string, returning nil for an empty base64 string.
func NewBytes(b64 string) Bytes {
	if b64 == "" {
		return Bytes(nil)
	}
	bs, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic(fmt.Sprintf("Invalid base64 string %q", b64))
	}
	return Bytes(bs)
}
