package resources

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
)

type Credentials struct {
	AWS    AWSCredentials
	Packet PacketCredentials
}

type AWSCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

type PacketCredentials struct {
	APIKey    string
	ProjectID string
}

type CredentialsData interface {
	Cluster() *kubermaticv1.Cluster
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
}

func GetCredentials(data CredentialsData) (Credentials, error) {
	credentials := Credentials{}
	var err error

	if data.Cluster().Spec.Cloud.AWS != nil {
		if credentials.AWS, err = GetAWSCredentials(data); err != nil {
			return Credentials{}, err
		}
	}
	if data.Cluster().Spec.Cloud.Packet != nil {
		if credentials.Packet, err = GetPacketCredentials(data); err != nil {
			return Credentials{}, err
		}
	}

	return credentials, err
}

func GetAWSCredentials(data CredentialsData) (AWSCredentials, error) {
	spec := data.Cluster().Spec.Cloud.AWS
	awsCredentials := AWSCredentials{}
	var err error

	if spec.AccessKeyID != "" {
		awsCredentials.AccessKeyID = spec.AccessKeyID
	} else {
		if awsCredentials.AccessKeyID, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AWSAccessKeyID); err != nil {
			return AWSCredentials{}, err
		}
	}

	if spec.SecretAccessKey != "" {
		awsCredentials.SecretAccessKey = spec.SecretAccessKey
	} else {
		if awsCredentials.SecretAccessKey, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, AWSSecretAccessKey); err != nil {
			return AWSCredentials{}, err
		}
	}

	return awsCredentials, nil
}

func GetPacketCredentials(data CredentialsData) (PacketCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Packet
	packetCredentials := PacketCredentials{}
	var err error

	if spec.APIKey != "" {
		packetCredentials.APIKey = spec.APIKey
	} else {
		if packetCredentials.APIKey, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, PacketAPIKey); err != nil {
			return PacketCredentials{}, err
		}
	}

	if spec.ProjectID != "" {
		packetCredentials.ProjectID = spec.ProjectID
	} else {
		if packetCredentials.ProjectID, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, PacketProjectID); err != nil {
			return PacketCredentials{}, err
		}
	}

	return packetCredentials, nil
}
