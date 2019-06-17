package utils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

func GetOIDCClient() (string, string) {
	user := os.Getenv("OIDC_USER")
	password := os.Getenv("OIDC_PASSWORD")
	if len(user) > 0 && len(password) > 0 {
		return user, password
	}

	return "roxy@loodse.com", "password"
}

func GetOIDCReqToken(hClient *http.Client, u url.URL, redirectURI string) (string, error) {
	u.Path = "auth"
	qp := u.Query()
	qp.Set("client_id", "kubermatic")
	qp.Set("redirect_uri", redirectURI)
	qp.Set("response_type", "code")
	qp.Set("scope", "openid profile email")
	qp.Set("state", "I wish to wash my irish wristwatch")
	u.RawQuery = qp.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", err
	}

	rsp, err := hClient.Do(req)
	if err != nil {
		return "", err
	}

	defer rsp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return "", err
	}

	body := string(bodyBytes)

	tokenPrefix := `href=\"\/auth\/local\?req=(\w)*"`
	tokenCompiledRegExp := regexp.MustCompile(tokenPrefix)
	potentialTokenLines := tokenCompiledRegExp.FindAllString(body, -1)
	if len(potentialTokenLines) < 1 {
		return "", fmt.Errorf("the response doesn't contain the expected text, the regular expression that was used=%q, unable to get a token from ODIC provider, full body=\n\n%q", tokenPrefix, body)
	}

	tokens := strings.Split(potentialTokenLines[0], "req=")
	if len(tokens) < 1 {
		return "", fmt.Errorf("unable to find a token, tried to split the text=%q by %q", potentialTokenLines[0], "req=")
	}
	token := tokens[1]
	return strings.TrimSuffix(token, "\""), nil
}

func GetOIDCAuthToken(hClient *http.Client, reqToken string, u url.URL, login string, password string) (string, error) {
	u.Path = "auth/local"
	qp := u.Query()
	qp.Set("req", reqToken)
	u.RawQuery = qp.Encode()

	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	if err := writer.WriteField("login", login); err != nil {
		return "", err
	}
	if err := writer.WriteField("password", password); err != nil {
		return "", err
	}
	err := writer.Close()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", u.String(), bytes.NewReader(buf.Bytes()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rsp, err := hClient.Do(req)
	if err != nil {
		return "", err
	}

	defer rsp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return "", err
	}

	tokenPrefix := `(\<code\>)(\s*.*\s*)(\<\/code\>)`
	tokenCompiledRegExp := regexp.MustCompile(tokenPrefix)
	body := string(bodyBytes)
	potentialTokenLines := tokenCompiledRegExp.FindAllString(body, -1)
	if len(potentialTokenLines) < 1 {
		return "", fmt.Errorf("the response doesn't contain the expected an OIDC token, the regular expression that was used=%q, unable to get a token from ODIC provider, full body=\n\n%q", tokenPrefix, body)
	}
	token := potentialTokenLines[0]
	token = strings.TrimPrefix(token, "<code>")
	token = strings.TrimSuffix(token, "</code>")

	return token, nil
}
