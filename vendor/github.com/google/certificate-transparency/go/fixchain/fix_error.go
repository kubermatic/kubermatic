package fixchain

import (
	"fmt"

	"github.com/google/certificate-transparency/go/x509"
)

type errorType int

// FixError types
const (
	None errorType = iota
	ParseFailure
	CannotFetchURL
	FixFailed
	LogPostFailed
	VerifyFailed
)

// FixError is the struct with which errors in the fixing process are reported
type FixError struct {
	Type  errorType
	Cert  *x509.Certificate   // The supplied leaf certificate
	Chain []*x509.Certificate // The supplied chain
	URL   string              // URL, if a URL is involved
	Bad   []byte              // The offending bytes, if applicable
	Error error               // And the error
}

// TypeString returns a string describing e.Type
func (e FixError) TypeString() string {
	switch e.Type {
	case None:
		return "None"
	case ParseFailure:
		return "ParseFailure"
	case CannotFetchURL:
		return "CannotFetchURL"
	case FixFailed:
		return "FixFailed"
	case LogPostFailed:
		return "LogPostFailed"
	case VerifyFailed:
		return "VerifyFailed"
	default:
		return fmt.Sprintf("Type %d", e.Type)
	}
}
