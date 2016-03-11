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

// MarshalJSON adds base64 json encoding to the Key type.
func (k Key) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", base64.StdEncoding.EncodeToString(k))), nil
}

// MarshalJSON adds base64 json encoding to the Cert type.
func (c Cert) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", base64.StdEncoding.EncodeToString(c))), nil
}

func unmarshalJSON(dest *[]byte, src []byte) error {
	if len(src) < 2 {
		return errors.New("base64 string expected")
	}
	if src[0] != '"' || src[len(src)-1] != '"' {
		return errors.New("\" quotations expected")
	}
	if len(src) == 2 {
		*dest = nil
		return nil
	}
	bs, err := base64.StdEncoding.DecodeString(string(src[1 : len(src)-1]))
	if err != nil {
		return err
	}
	*dest = bs
	return nil
}

// UnmarshalJSON adds base64 json decoding to the Key type.
func (k Key) UnmarshalJSON(bs []byte) error {
	var dest []byte
	err := unmarshalJSON(&dest, bs)
	if err != nil {
		return err
	}
	k = Key(dest)
	_ = k
	return nil
}

// UnmarshalJSON adds base64 json decoding to the Cert type.
func (c Cert) UnmarshalJSON(bs []byte) error {
	var dest []byte
	err := unmarshalJSON(&dest, bs)
	if err != nil {
		return err
	}
	c = Cert(bs)
	_ = c
	return nil
}
