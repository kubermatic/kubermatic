//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2023 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package aws

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/vmware-tanzu/velero/pkg/plugin/framework"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
)

type awsBackupStore struct {
	provider              string
	bucket                string
	prefix                string
	caCert                []byte
	s3Url                 string
	region                string
	profile               string
	insecureSkipTLSVerify bool
	s3ForcePathStyle      bool
	credentials           *corev1.Secret
}

const (
	AccessKeyIDKeyName  = "accessKeyId"
	SecretAccessKeyName = "secretAccessKey"
)

var validVeleroConfigKeys = []string{
	"bucket",
	"caCert",
	"credentialsFile",
	"customerKeyEncryptionFile",
	"enableSharedConfig",
	"insecureSkipTLSVerify",
	"kmsKeyId",
	"profile",
	"publicUrl",
	"region",
	"s3ForcePathStyle",
	"s3Url",
	"serverSideEncryption",
	"signatureVersion",
	"tagging",
}

func NewBackupStore(cbsl *kubermaticv1.ClusterBackupStorageLocation, credentials *corev1.Secret) (*awsBackupStore, error) {
	spec := cbsl.Spec
	err := validateVeleroConfig(cbsl)
	if err != nil {
		return nil, fmt.Errorf("invalid backup storage configuration: %w", err)
	}

	var skipTLS, s3ForcePathStyle bool
	if _, ok := spec.Config["insecureSkipTLSVerify"]; ok {
		skipTLS, err = strconv.ParseBool(spec.Config["insecureSkipTLSVerify"])
		if err != nil {
			return nil, fmt.Errorf("failed to parse value: %s", spec.Config["insecureSkipTLSVerify"])
		}
	}
	if _, ok := spec.Config["s3ForcePathStyle"]; ok {
		s3ForcePathStyle, err = strconv.ParseBool(spec.Config["s3ForcePathStyle"])
		if err != nil {
			return nil, fmt.Errorf("failed to parse value: %s", spec.Config["s3ForcePathStyle"])
		}
	}

	if credentials == nil {
		return nil, fmt.Errorf("invalid backup storage configuration: credentials secret can't be empty")
	}
	return &awsBackupStore{
		provider:              cbsl.Spec.Provider,
		bucket:                spec.ObjectStorage.Bucket,
		prefix:                spec.ObjectStorage.Prefix,
		caCert:                spec.ObjectStorage.CACert,
		s3Url:                 spec.Config["s3Url"],
		region:                spec.Config["region"],
		profile:               spec.Config["profile"],
		insecureSkipTLSVerify: skipTLS,
		s3ForcePathStyle:      s3ForcePathStyle,
		credentials:           credentials,
	}, nil
}

func validateVeleroConfig(cbsl *kubermaticv1.ClusterBackupStorageLocation) error {
	if err := framework.ValidateObjectStoreConfigKeys(cbsl.Spec.Config, validVeleroConfigKeys...); err != nil {
		return fmt.Errorf("invalid object store config: %w", err)
	}
	if cbsl.Spec.ObjectStorage == nil {
		return errors.New("ObjectStorage can't be empty")
	}
	if cbsl.Spec.ObjectStorage.Bucket == "" {
		return errors.New("bucket can't be empty")
	}
	return nil
}

func (p *awsBackupStore) IsValid(ctx context.Context) error {
	s3Client, err := p.newS3Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}
	if _, err = s3Client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &p.bucket}); err != nil {
		return fmt.Errorf("failed to access bucket [%s]: %w", p.bucket, err)
	}
	return nil
}

func (p *awsBackupStore) readAWSCredentials() (accessKeyID, secretKey string, err error) {
	if p.credentials == nil || p.credentials.Data == nil {
		return "", "", errors.New("can't read AWS credentials: empty secret object")
	}

	id, ok := p.credentials.Data[AccessKeyIDKeyName]
	if !ok {
		return "", "", fmt.Errorf("can't read AWS credentials: %s is not set", AccessKeyIDKeyName)
	}
	secret, ok := p.credentials.Data[SecretAccessKeyName]
	if !ok {
		return "", "", fmt.Errorf("can't read AWS credentials: %s is not set", SecretAccessKeyName)
	}
	return string(id), string(secret), nil
}

func customHTTPClient(caCert []byte, skipTLSVerify bool) *http.Client {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipTLSVerify,
			RootCAs:            caCertPool,
		},
	}
	return &http.Client{Transport: tr}
}

func (p *awsBackupStore) newS3Client(ctx context.Context) (*s3.Client, error) {
	awsOptions := []func(*config.LoadOptions) error{
		config.WithRegion(p.region),
		config.WithSharedConfigProfile(p.profile),
	}

	if p.caCert != nil || p.insecureSkipTLSVerify {
		awsOptions = append(awsOptions, config.WithHTTPClient(customHTTPClient(p.caCert, p.insecureSkipTLSVerify)))
	}

	accessKeyID, secretKey, err := p.readAWSCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}
	awsOptions = append(awsOptions, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, "")))

	awsConfig, err := config.LoadDefaultConfig(ctx, awsOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to load default AWS config: %w", err)
	}

	s3Options := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = p.s3ForcePathStyle
		},
	}

	if p.s3Url != "" {
		if err := validateS3URL(p.s3Url); err != nil {
			return nil, fmt.Errorf("invalid S3 URL: %w", err)
		}
		s3Options = append(s3Options, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(p.s3Url)
		})
	}
	return s3.NewFromConfig(awsConfig, s3Options...), nil
}

func validateS3URL(s3Url string) error {
	parsedURL, err := url.Parse(s3Url)
	if err != nil {
		return fmt.Errorf("failed to parse s3Url: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("invalid s3Url scheme. Only 'http' or 'https' are supported")
	}

	return nil
}
