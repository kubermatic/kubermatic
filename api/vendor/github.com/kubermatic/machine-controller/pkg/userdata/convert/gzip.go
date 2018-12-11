package convert

import (
	"bytes"
	"compress/gzip"
)

func GzipString(s string) (string, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)

	if _, err := gz.Write([]byte(s)); err != nil {
		return "", err
	}

	if err := gz.Flush(); err != nil {
		return "", err
	}

	if err := gz.Close(); err != nil {
		return "", err
	}

	return b.String(), nil
}
