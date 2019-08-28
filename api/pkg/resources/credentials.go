package resources

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
)

type Credentials struct {
	AWS          AWSCredentials
	Digitalocean DigitaloceanCredentials
	Hetzner      HetznerCredentials
	Packet       PacketCredentials
}

type AWSCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

type DigitaloceanCredentials struct {
	Token string
}

type HetznerCredentials struct {
	Token string
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
	if data.Cluster().Spec.Cloud.Digitalocean != nil {
		if credentials.Digitalocean, err = GetDigitaloceanCredentials(data); err != nil {
			return Credentials{}, err
		}
	}
	if data.Cluster().Spec.Cloud.Hetzner != nil {
		if credentials.Hetzner, err = GetHetznerCredentials(data); err != nil {
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

func GetDigitaloceanCredentials(data CredentialsData) (DigitaloceanCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Digitalocean
	digitaloceanCredentials := DigitaloceanCredentials{}
	var err error

	if spec.Token != "" {
		digitaloceanCredentials.Token = spec.Token
	} else {
		if digitaloceanCredentials.Token, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, DigitaloceanToken); err != nil {
			return DigitaloceanCredentials{}, err
		}
	}

	return digitaloceanCredentials, nil
}

func GetHetznerCredentials(data CredentialsData) (HetznerCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Hetzner
	hetznerCredentials := HetznerCredentials{}
	var err error

	if spec.Token != "" {
		hetznerCredentials.Token = spec.Token
	} else {
		if hetznerCredentials.Token, err = data.GetGlobalSecretKeySelectorValue(spec.CredentialsReference, HetznerToken); err != nil {
			return HetznerCredentials{}, err
		}
	}

	return hetznerCredentials, nil
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
