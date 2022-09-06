/*
Copyright 2017-2020 by the contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package authenticator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/endpointcreds"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	smithymiddleware "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/apis/clientauthentication"
	clientauthv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
	"k8s.io/utils/pointer"
)

// Identity is returned on successful Verify() results. It contains a parsed
// version of the AWS identity used to create the token.
type Identity struct {
	// ARN is the raw Amazon Resource Name returned by sts:GetCallerIdentity
	ARN string

	// CanonicalARN is the Amazon Resource Name converted to a more canonical
	// representation. In particular, STS assumed role ARNs like
	// "arn:aws:sts::ACCOUNTID:assumed-role/ROLENAME/SESSIONNAME" are converted
	// to their IAM ARN equivalent "arn:aws:iam::ACCOUNTID:role/NAME"
	CanonicalARN string

	// AccountID is the 12 digit AWS account number.
	AccountID string

	// UserID is the unique user/role ID (e.g., "AROAAAAAAAAAAAAAAAAAA").
	UserID string

	// SessionName is the STS session name (or "" if this is not a
	// session-based identity). For EC2 instance roles, this will be the EC2
	// instance ID (e.g., "i-0123456789abcdef0"). You should only rely on it
	// if you trust that _only_ EC2 is allowed to assume the IAM Role. If IAM
	// users or other roles are allowed to assume the role, they can provide
	// (nearly) arbitrary strings here.
	SessionName string

	// The AWS Access Key ID used to authenticate the request.  This can be used
	// in conjunction with CloudTrail to determine the identity of the individual
	// if the individual assumed an IAM role before making the request.
	AccessKeyID string
}

const (
	// The sts GetCallerIdentity request is valid for 15 minutes regardless of this parameters value after it has been
	// signed, but we set this unused parameter to 60 for legacy reasons (we check for a value between 0 and 60 on the
	// server side in 0.3.0 or earlier).  IT IS IGNORED.  If we can get STS to support x-amz-expires, then we should
	// set this parameter to the actual expiration, and make it configurable.
	requestPresignParam = 60
	// The actual token expiration (presigned STS urls are valid for 15 minutes after timestamp in x-amz-date).
	presignedURLExpiration = 15 * time.Minute
	v1Prefix               = "k8s-aws-v1."
	maxTokenLenBytes       = 1024 * 4
	clusterIDHeader        = "x-k8s-aws-id"
	// Format of the X-Amz-Date header used for expiration
	// https://golang.org/pkg/time/#pkg-constants
	dateHeaderFormat   = "20060102T150405Z"
	kindExecCredential = "ExecCredential"
	execInfoEnvKey     = "KUBERNETES_EXEC_INFO"
)

// Token is generated and used by Kubernetes client-go to authenticate with a Kubernetes cluster.
type Token struct {
	Token      string
	Expiration time.Time
}

// GetTokenOptions is passed to GetWithOptions to provide an extensible get token interface
type GetTokenOptions struct {
	Region               string
	ClusterID            string
	AssumeRoleARN        string
	AssumeRoleExternalID string
	SessionName          string
	Config               *aws.Config
}

// FormatError is returned when there is a problem with token that is
// an encoded sts request.  This can include the url, data, action or anything
// else that prevents the sts call from being made.
type FormatError struct {
	message string
}

func (e FormatError) Error() string {
	return "input token was not properly formatted: " + e.message
}

// STSError is returned when there was either an error calling STS or a problem
// processing the data returned from STS.
type STSError struct {
	message string
}

func (e STSError) Error() string {
	return "sts getCallerIdentity failed: " + e.message
}

// NewSTSError creates a error of type STS.
func NewSTSError(m string) STSError {
	return STSError{message: m}
}

var parameterWhitelist = map[string]bool{
	"action":               true,
	"version":              true,
	"x-amz-algorithm":      true,
	"x-amz-credential":     true,
	"x-amz-date":           true,
	"x-amz-expires":        true,
	"x-amz-security-token": true,
	"x-amz-signature":      true,
	"x-amz-signedheaders":  true,
}

// this is the result type from the GetCallerIdentity endpoint
type getCallerIdentityWrapper struct {
	GetCallerIdentityResponse struct {
		GetCallerIdentityResult struct {
			Account string `json:"Account"`
			Arn     string `json:"Arn"`
			UserID  string `json:"UserId"`
		} `json:"GetCallerIdentityResult"`
		ResponseMetadata struct {
			RequestID string `json:"RequestId"`
		} `json:"ResponseMetadata"`
	} `json:"GetCallerIdentityResponse"`
}

// Generator provides new tokens for the AWS IAM Authenticator.
type Generator interface {
	// Get a token using credentials in the default credentials chain.
	Get(ctx context.Context, clusterID string) (Token, error)
	// GetWithRole creates a token by assuming the provided role, using the credentials in the default chain.
	GetWithRole(ctx context.Context, clusterID, roleARN string) (Token, error)
	// GetWithRoleForSession creates a token by assuming the provided role, using the provided configuration.
	GetWithRoleForSession(ctx context.Context, clusterID string, roleARN string, cfg *aws.Config) (Token, error)
	// Get a token using the provided options
	GetWithOptions(ctx context.Context, options *GetTokenOptions) (Token, error)
	// GetWithSTS returns a token valid for clusterID using the given STS client.
	GetWithSTS(ctx context.Context, clusterID string, stsAPI *sts.Client) (Token, error)
	// FormatJSON returns the client auth formatted json for the ExecCredential auth
	FormatJSON(Token) string
}

type generator struct {
	forwardSessionName bool
}

// verify interface at compile time
var _ Generator = &generator{}

// NewGenerator creates a Generator and returns it.
func NewGenerator(forwardSessionName bool) (Generator, error) {
	return generator{
		forwardSessionName: forwardSessionName,
	}, nil
}

// Get uses the directly available AWS credentials to return a token valid for
// clusterID. It follows the default AWS credential handling behavior.
func (g generator) Get(ctx context.Context, clusterID string) (Token, error) {
	return g.GetWithOptions(ctx, &GetTokenOptions{ClusterID: clusterID})
}

// GetWithRole assumes the given AWS IAM role and returns a token valid for
// clusterID. If roleARN is empty, behaves like Get (does not assume a role).
func (g generator) GetWithRole(ctx context.Context, clusterID string, roleARN string) (Token, error) {
	return g.GetWithOptions(ctx, &GetTokenOptions{
		ClusterID:     clusterID,
		AssumeRoleARN: roleARN,
	})
}

// GetWithRoleForSession assumes the given AWS IAM role for the given session and behaves
// like GetWithRole.
func (g generator) GetWithRoleForSession(ctx context.Context, clusterID string, roleARN string, cfg *aws.Config) (Token, error) {
	return g.GetWithOptions(ctx, &GetTokenOptions{
		ClusterID:     clusterID,
		AssumeRoleARN: roleARN,
		Config:        cfg,
	})
}

// StdinStderrTokenProvider gets MFA token from standard input.
func StdinStderrTokenProvider() (string, error) {
	var v string
	fmt.Fprint(os.Stderr, "Assume Role MFA token code: ")
	_, err := fmt.Scanln(&v)
	return v, err
}

// GetWithOptions takes a GetTokenOptions struct, builds the STS client, and wraps GetWithSTS.
// If no session has been passed in options, it will build a new session. If an
// AssumeRoleARN was passed in then assume the role for the session.
func (g generator) GetWithOptions(ctx context.Context, options *GetTokenOptions) (Token, error) {
	if options.ClusterID == "" {
		return Token{}, fmt.Errorf("ClusterID is required")
	}

	if options.Config == nil {
		var loadOptFuncs []func(*awsconfig.LoadOptions) error

		loadOptFuncs = append(loadOptFuncs, awsconfig.WithAPIOptions([]func(*smithymiddleware.Stack) error{
			smithyhttp.AddHeaderValue("User-Agent", "aws-iam-authenticator/v0.5.8-forked"),
		}))

		loadOptFuncs = append(loadOptFuncs, awsconfig.WithAssumeRoleCredentialOptions(func(opts *stscreds.AssumeRoleOptions) {
			opts.TokenProvider = StdinStderrTokenProvider
		}))

		if options.Region != "" {
			loadOptFuncs = append(loadOptFuncs, awsconfig.WithRegion(options.Region))

			endpoint, err := sts.NewDefaultEndpointResolver().ResolveEndpoint(options.Region, sts.EndpointResolverOptions{})
			if err != nil {
				return Token{}, err
			}

			loadOptFuncs = append(loadOptFuncs, awsconfig.WithEndpointCredentialOptions(func(opts *endpointcreds.Options) {
				opts.Endpoint = endpoint.URL
			}))
		}

		cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptFuncs...)
		if err != nil {
			return Token{}, fmt.Errorf("failed to get AWS config: %w", err)
		}

		options.Config = &cfg
	}

	// use an STS client based on the direct credentials
	stsAPI := sts.NewFromConfig(*options.Config)

	// if a roleARN was specified, replace the STS client with one that uses
	// temporary credentials from that role.
	if options.AssumeRoleARN != "" {
		var assumeOptFuncs []func(*stscreds.AssumeRoleOptions)

		if options.AssumeRoleExternalID != "" {
			assumeOptFuncs = append(assumeOptFuncs, func(opts *stscreds.AssumeRoleOptions) {
				opts.ExternalID = pointer.String(options.AssumeRoleExternalID)
			})
		}

		if g.forwardSessionName {
			// If the current session is already a federated identity, carry through
			// this session name onto the new session to provide better debugging
			// capabilities
			resp, err := stsAPI.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
			if err != nil {
				return Token{}, err
			}

			userIDParts := strings.Split(*resp.UserId, ":")

			assumeOptFuncs = append(assumeOptFuncs, func(opts *stscreds.AssumeRoleOptions) {
				opts.RoleARN = userIDParts[1]
			})

		} else if options.SessionName != "" {
			assumeOptFuncs = append(assumeOptFuncs, func(opts *stscreds.AssumeRoleOptions) {
				opts.RoleSessionName = options.SessionName
			})
		}

		cfg := options.Config.Copy()
		cfg.Credentials = stscreds.NewAssumeRoleProvider(stsAPI, options.AssumeRoleARN, assumeOptFuncs...)

		// create an STS API interface that uses the assumed role's temporary credentials
		stsAPI = sts.NewFromConfig(cfg)
	}

	return g.GetWithSTS(ctx, options.ClusterID, stsAPI)
}

// GetWithSTS returns a token valid for clusterID using the given STS client.
func (g generator) GetWithSTS(ctx context.Context, clusterID string, stsAPI *sts.Client) (Token, error) {
	presignClient := sts.NewPresignClient(stsAPI)
	presignedURLRequest, err := presignClient.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}, func(presignOpts *sts.PresignOptions) {
		presignOpts.ClientOptions = append(presignOpts.ClientOptions, func(stsOpts *sts.Options) {
			stsOpts.APIOptions = append(stsOpts.APIOptions, smithyhttp.SetHeaderValue(clusterIDHeader, clusterID))
		})
	})

	if err != nil {
		return Token{}, err
	}

	// Set token expiration to 1 minute before the presigned URL expires for some cushion
	tokenExpiration := time.Now().Local().Add(presignedURLExpiration - 1*time.Minute)
	// TODO: this may need to be a constant-time base64 encoding
	return Token{v1Prefix + base64.RawURLEncoding.EncodeToString([]byte(presignedURLRequest.URL)), tokenExpiration}, nil
}

// FormatJSON formats the json to support ExecCredential authentication
func (g generator) FormatJSON(token Token) string {
	apiVersion := clientauthv1beta1.SchemeGroupVersion.String()
	env := os.Getenv(execInfoEnvKey)
	if env != "" {
		cred := &clientauthentication.ExecCredential{}
		if err := json.Unmarshal([]byte(env), cred); err == nil {
			apiVersion = cred.APIVersion
		}
	}

	expirationTimestamp := metav1.NewTime(token.Expiration)
	execInput := &clientauthv1beta1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       kindExecCredential,
		},
		Status: &clientauthv1beta1.ExecCredentialStatus{
			ExpirationTimestamp: &expirationTimestamp,
			Token:               token.Token,
		},
	}
	enc, _ := json.Marshal(execInput)
	return string(enc)
}
