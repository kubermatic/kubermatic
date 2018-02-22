package resources

import (
	"bytes"
	"encoding/csv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

// Token represents a user token in a kubernetes cluster
type Token struct {
	Token  string
	Name   string
	UserID string
	Group  string
}

// GenerateTokenCSV returns a secret with the given user tokens
func GenerateTokenCSV(name string, tokens []Token) (*corev1.Secret, string, error) {
	buffer := bytes.Buffer{}
	writer := csv.NewWriter(&buffer)

	for _, token := range tokens {
		if err := writer.Write([]string{token.Token, token.Name, token.UserID, token.Group}); err != nil {
			return nil, "", err
		}
	}
	writer.Flush()
	b := buffer.Bytes()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"tokens.csv": b,
		},
	}

	j, err := json.Marshal(secret)
	if err != nil {
		return nil, "", err
	}

	return secret, string(j), nil

}
