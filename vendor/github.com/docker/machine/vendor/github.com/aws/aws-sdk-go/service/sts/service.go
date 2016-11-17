// THIS FILE IS AUTOMATICALLY GENERATED. DO NOT EDIT.

package sts

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/private/protocol/query"
)

// The AWS Security Token Service (STS) is a web service that enables you to
// request temporary, limited-privilege credentials for AWS Identity and Access
// Management (IAM) users or for users that you authenticate (federated users).
// This guide provides descriptions of the STS API. For more detailed information
// about using this service, go to Temporary Security Credentials (http://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html).
//
//   As an alternative to using the API, you can use one of the AWS SDKs, which
// consist of libraries and sample code for various programming languages and
// platforms (Java, Ruby, .NET, iOS, Android, etc.). The SDKs provide a convenient
// way to create programmatic access to STS. For example, the SDKs take care
// of cryptographically signing requests, managing errors, and retrying requests
// automatically. For information about the AWS SDKs, including how to download
// and install them, see the Tools for Amazon Web Services page (http://aws.amazon.com/tools/).
//
//  For information about setting up signatures and authorization through the
// API, go to Signing AWS API Requests (http://docs.aws.amazon.com/general/latest/gr/signing_aws_api_requests.html)
// in the AWS General Reference. For general information about the Query API,
// go to Making Query Requests (http://docs.aws.amazon.com/IAM/latest/UserGuide/IAM_UsingQueryAPI.html)
// in Using IAM. For information about using security tokens with other AWS
// products, go to AWS Services That Work with IAM (http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_aws-services-that-work-with-iam.html)
// in the IAM User Guide.
//
// If you're new to AWS and need additional technical information about a specific
// AWS product, you can find the product's technical documentation at http://aws.amazon.com/documentation/
// (http://aws.amazon.com/documentation/).
//
//  Endpoints
//
// The AWS Security Token Service (STS) has a default endpoint of https://sts.amazonaws.com
// that maps to the US East (N. Virginia) region. Additional regions are available
// and are activated by default. For more information, see Activating and Deactivating
// AWS STS in an AWS Region (http://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_enable-regions.html)
// in the IAM User Guide.
//
// For information about STS endpoints, see Regions and Endpoints (http://docs.aws.amazon.com/general/latest/gr/rande.html#sts_region)
// in the AWS General Reference.
//
//  Recording API requests
//
// STS supports AWS CloudTrail, which is a service that records AWS calls for
// your AWS account and delivers log files to an Amazon S3 bucket. By using
// information collected by CloudTrail, you can determine what requests were
// successfully made to STS, who made the request, when it was made, and so
// on. To learn more about CloudTrail, including how to turn it on and find
// your log files, see the AWS CloudTrail User Guide (http://docs.aws.amazon.com/awscloudtrail/latest/userguide/what_is_cloud_trail_top_level.html).
//The service client's operations are safe to be used concurrently.
// It is not safe to mutate any of the client's properties though.
type STS struct {
	*client.Client
}

// Used for custom client initialization logic
var initClient func(*client.Client)

// Used for custom request initialization logic
var initRequest func(*request.Request)

// A ServiceName is the name of the service the client will make API calls to.
const ServiceName = "sts"

// New creates a new instance of the STS client with a session.
// If additional configuration is needed for the client instance use the optional
// aws.Config parameter to add your extra config.
//
// Example:
//     // Create a STS client from just a session.
//     svc := sts.New(mySession)
//
//     // Create a STS client with additional configuration
//     svc := sts.New(mySession, aws.NewConfig().WithRegion("us-west-2"))
func New(p client.ConfigProvider, cfgs ...*aws.Config) *STS {
	c := p.ClientConfig(ServiceName, cfgs...)
	return newClient(*c.Config, c.Handlers, c.Endpoint, c.SigningRegion)
}

// newClient creates, initializes and returns a new service client instance.
func newClient(cfg aws.Config, handlers request.Handlers, endpoint, signingRegion string) *STS {
	svc := &STS{
		Client: client.New(
			cfg,
			metadata.ClientInfo{
				ServiceName:   ServiceName,
				SigningRegion: signingRegion,
				Endpoint:      endpoint,
				APIVersion:    "2011-06-15",
			},
			handlers,
		),
	}

	// Handlers
	svc.Handlers.Sign.PushBackNamed(v4.SignRequestHandler)
	svc.Handlers.Build.PushBackNamed(query.BuildHandler)
	svc.Handlers.Unmarshal.PushBackNamed(query.UnmarshalHandler)
	svc.Handlers.UnmarshalMeta.PushBackNamed(query.UnmarshalMetaHandler)
	svc.Handlers.UnmarshalError.PushBackNamed(query.UnmarshalErrorHandler)

	// Run custom client initialization if present
	if initClient != nil {
		initClient(svc.Client)
	}

	return svc
}

// newRequest creates a new request for a STS operation and runs any
// custom request initialization.
func (c *STS) newRequest(op *request.Operation, params, data interface{}) *request.Request {
	req := c.NewRequest(op, params, data)

	// Run custom request initialization if present
	if initRequest != nil {
		initRequest(req)
	}

	return req
}
