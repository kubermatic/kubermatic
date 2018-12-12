// Code generated by private/model/cli/gen-api/main.go. DO NOT EDIT.

package iam

const (

	// ErrCodeConcurrentModificationException for service response error code
	// "ConcurrentModification".
	//
	// The request was rejected because multiple requests to change this object
	// were submitted simultaneously. Wait a few minutes and submit your request
	// again.
	ErrCodeConcurrentModificationException = "ConcurrentModification"

	// ErrCodeCredentialReportExpiredException for service response error code
	// "ReportExpired".
	//
	// The request was rejected because the most recent credential report has expired.
	// To generate a new credential report, use GenerateCredentialReport. For more
	// information about credential report expiration, see Getting Credential Reports
	// (http://docs.aws.amazon.com/IAM/latest/UserGuide/credential-reports.html)
	// in the IAM User Guide.
	ErrCodeCredentialReportExpiredException = "ReportExpired"

	// ErrCodeCredentialReportNotPresentException for service response error code
	// "ReportNotPresent".
	//
	// The request was rejected because the credential report does not exist. To
	// generate a credential report, use GenerateCredentialReport.
	ErrCodeCredentialReportNotPresentException = "ReportNotPresent"

	// ErrCodeCredentialReportNotReadyException for service response error code
	// "ReportInProgress".
	//
	// The request was rejected because the credential report is still being generated.
	ErrCodeCredentialReportNotReadyException = "ReportInProgress"

	// ErrCodeDeleteConflictException for service response error code
	// "DeleteConflict".
	//
	// The request was rejected because it attempted to delete a resource that has
	// attached subordinate entities. The error message describes these entities.
	ErrCodeDeleteConflictException = "DeleteConflict"

	// ErrCodeDuplicateCertificateException for service response error code
	// "DuplicateCertificate".
	//
	// The request was rejected because the same certificate is associated with
	// an IAM user in the account.
	ErrCodeDuplicateCertificateException = "DuplicateCertificate"

	// ErrCodeDuplicateSSHPublicKeyException for service response error code
	// "DuplicateSSHPublicKey".
	//
	// The request was rejected because the SSH public key is already associated
	// with the specified IAM user.
	ErrCodeDuplicateSSHPublicKeyException = "DuplicateSSHPublicKey"

	// ErrCodeEntityAlreadyExistsException for service response error code
	// "EntityAlreadyExists".
	//
	// The request was rejected because it attempted to create a resource that already
	// exists.
	ErrCodeEntityAlreadyExistsException = "EntityAlreadyExists"

	// ErrCodeEntityTemporarilyUnmodifiableException for service response error code
	// "EntityTemporarilyUnmodifiable".
	//
	// The request was rejected because it referenced an entity that is temporarily
	// unmodifiable, such as a user name that was deleted and then recreated. The
	// error indicates that the request is likely to succeed if you try again after
	// waiting several minutes. The error message describes the entity.
	ErrCodeEntityTemporarilyUnmodifiableException = "EntityTemporarilyUnmodifiable"

	// ErrCodeInvalidAuthenticationCodeException for service response error code
	// "InvalidAuthenticationCode".
	//
	// The request was rejected because the authentication code was not recognized.
	// The error message describes the specific error.
	ErrCodeInvalidAuthenticationCodeException = "InvalidAuthenticationCode"

	// ErrCodeInvalidCertificateException for service response error code
	// "InvalidCertificate".
	//
	// The request was rejected because the certificate is invalid.
	ErrCodeInvalidCertificateException = "InvalidCertificate"

	// ErrCodeInvalidInputException for service response error code
	// "InvalidInput".
	//
	// The request was rejected because an invalid or out-of-range value was supplied
	// for an input parameter.
	ErrCodeInvalidInputException = "InvalidInput"

	// ErrCodeInvalidPublicKeyException for service response error code
	// "InvalidPublicKey".
	//
	// The request was rejected because the public key is malformed or otherwise
	// invalid.
	ErrCodeInvalidPublicKeyException = "InvalidPublicKey"

	// ErrCodeInvalidUserTypeException for service response error code
	// "InvalidUserType".
	//
	// The request was rejected because the type of user for the transaction was
	// incorrect.
	ErrCodeInvalidUserTypeException = "InvalidUserType"

	// ErrCodeKeyPairMismatchException for service response error code
	// "KeyPairMismatch".
	//
	// The request was rejected because the public key certificate and the private
	// key do not match.
	ErrCodeKeyPairMismatchException = "KeyPairMismatch"

	// ErrCodeLimitExceededException for service response error code
	// "LimitExceeded".
	//
	// The request was rejected because it attempted to create resources beyond
	// the current AWS account limits. The error message describes the limit exceeded.
	ErrCodeLimitExceededException = "LimitExceeded"

	// ErrCodeMalformedCertificateException for service response error code
	// "MalformedCertificate".
	//
	// The request was rejected because the certificate was malformed or expired.
	// The error message describes the specific error.
	ErrCodeMalformedCertificateException = "MalformedCertificate"

	// ErrCodeMalformedPolicyDocumentException for service response error code
	// "MalformedPolicyDocument".
	//
	// The request was rejected because the policy document was malformed. The error
	// message describes the specific error.
	ErrCodeMalformedPolicyDocumentException = "MalformedPolicyDocument"

	// ErrCodeNoSuchEntityException for service response error code
	// "NoSuchEntity".
	//
	// The request was rejected because it referenced an entity that does not exist.
	// The error message describes the entity.
	ErrCodeNoSuchEntityException = "NoSuchEntity"

	// ErrCodePasswordPolicyViolationException for service response error code
	// "PasswordPolicyViolation".
	//
	// The request was rejected because the provided password did not meet the requirements
	// imposed by the account password policy.
	ErrCodePasswordPolicyViolationException = "PasswordPolicyViolation"

	// ErrCodePolicyEvaluationException for service response error code
	// "PolicyEvaluation".
	//
	// The request failed because a provided policy could not be successfully evaluated.
	// An additional detailed message indicates the source of the failure.
	ErrCodePolicyEvaluationException = "PolicyEvaluation"

	// ErrCodePolicyNotAttachableException for service response error code
	// "PolicyNotAttachable".
	//
	// The request failed because AWS service role policies can only be attached
	// to the service-linked role for that service.
	ErrCodePolicyNotAttachableException = "PolicyNotAttachable"

	// ErrCodeServiceFailureException for service response error code
	// "ServiceFailure".
	//
	// The request processing has failed because of an unknown error, exception
	// or failure.
	ErrCodeServiceFailureException = "ServiceFailure"

	// ErrCodeServiceNotSupportedException for service response error code
	// "NotSupportedService".
	//
	// The specified service does not support service-specific credentials.
	ErrCodeServiceNotSupportedException = "NotSupportedService"

	// ErrCodeUnmodifiableEntityException for service response error code
	// "UnmodifiableEntity".
	//
	// The request was rejected because only the service that depends on the service-linked
	// role can modify or delete the role on your behalf. The error message includes
	// the name of the service that depends on this service-linked role. You must
	// request the change through that service.
	ErrCodeUnmodifiableEntityException = "UnmodifiableEntity"

	// ErrCodeUnrecognizedPublicKeyEncodingException for service response error code
	// "UnrecognizedPublicKeyEncoding".
	//
	// The request was rejected because the public key encoding format is unsupported
	// or unrecognized.
	ErrCodeUnrecognizedPublicKeyEncodingException = "UnrecognizedPublicKeyEncoding"
)
