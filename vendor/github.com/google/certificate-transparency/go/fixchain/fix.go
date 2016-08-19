package fixchain

import (
	"encoding/pem"
	"log"
	"net/http"

	"github.com/google/certificate-transparency/go/x509"
)

// Fix attempts to fix the certificate chain for the certificate that is passed
// to it, with respect to the given roots.  Fix returns a list of successfully
// constructed chains, and a list of errors it encountered along the way.  The
// presence of FixErrors does not mean the fix was unsuccessful.  Callers should
// check for returned chains to determine success.
func Fix(cert *x509.Certificate, chain []*x509.Certificate, roots *x509.CertPool, client *http.Client) ([][]*x509.Certificate, []*FixError) {
	fix := &toFix{
		cert:  cert,
		chain: newDedupedChain(chain),
		roots: roots,
		cache: newURLCache(client, false),
	}
	return fix.handleChain()
}

type toFix struct {
	cert  *x509.Certificate
	chain *dedupedChain
	roots *x509.CertPool
	opts  *x509.VerifyOptions
	cache *urlCache
}

func (fix *toFix) handleChain() ([][]*x509.Certificate, []*FixError) {
	intermediates := x509.NewCertPool()
	for _, c := range fix.chain.certs {
		intermediates.AddCert(c)
	}

	fix.opts = &x509.VerifyOptions{
		Intermediates:     intermediates,
		Roots:             fix.roots,
		DisableTimeChecks: true,
		KeyUsages:         []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	var retferrs []*FixError
	chains, ferrs := fix.constructChain()
	if ferrs != nil {
		retferrs = append(retferrs, ferrs...)
		chains, ferrs = fix.fixChain()
		if ferrs != nil {
			retferrs = append(retferrs, ferrs...)
		}
	}
	return chains, retferrs
}

func (fix *toFix) constructChain() ([][]*x509.Certificate, []*FixError) {
	chains, err := fix.cert.Verify(*fix.opts)
	if err != nil {
		return chains, []*FixError{
			&FixError{
				Type:  VerifyFailed,
				Cert:  fix.cert,
				Chain: fix.chain.certs,
				Error: err,
			},
		}
	}
	return chains, nil
}

func (fix *toFix) fixChain() ([][]*x509.Certificate, []*FixError) {
	var ferrs []*FixError
	d := *fix.chain
	d.addCert(fix.cert)
	for _, c := range d.certs {
		urls := c.IssuingCertificateURL
		for _, url := range urls {
			ferr := fix.augmentIntermediates(url)
			if ferr != nil {
				ferrs = append(ferrs, ferr)
			}
			chains, err := fix.cert.Verify(*fix.opts)
			if err == nil {
				return chains, nil
			}
		}
	}
	return nil, append(ferrs, &FixError{
		Type:  FixFailed,
		Cert:  fix.cert,
		Chain: fix.chain.certs,
	})
}

func (fix *toFix) augmentIntermediates(url string) *FixError {
	// PKCS#7 additions as (at time of writing) there is no standard Go PKCS#7
	// implementation
	r := urlReplacement(url)
	if r != nil {
		log.Printf("Replaced %s: %+v", url, r)
		for _, c := range r {
			fix.opts.Intermediates.AddCert(c)
		}
		return nil
	}

	body, err := fix.cache.getURL(url)
	if err != nil {
		return &FixError{
			Type:  CannotFetchURL,
			Cert:  fix.cert,
			Chain: fix.chain.certs,
			URL:   url,
			Error: err,
		}
	}
	icert, err := x509.ParseCertificate(body)
	if err != nil {
		s, _ := pem.Decode(body)
		if s != nil {
			icert, err = x509.ParseCertificate(s.Bytes)
		}
	}

	if err != nil {
		return &FixError{
			Type:  ParseFailure,
			Cert:  fix.cert,
			Chain: fix.chain.certs,
			URL:   url,
			Bad:   body,
			Error: err,
		}
	}
	fix.opts.Intermediates.AddCert(icert)
	return nil
}
