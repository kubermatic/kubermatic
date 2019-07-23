package machinecontroller

type Credentials struct {
	Packet PacketCredentials
}

type PacketCredentials struct {
	APIKey    string
	ProjectID string
}

func GetPacketCredentials(data machinecontrollerData) (PacketCredentials, error) {
	spec := data.Cluster().Spec.Cloud.Packet
	packetCredentials := PacketCredentials{}
	var err error

	if spec.APIKey != "" {
		packetCredentials.APIKey = spec.APIKey
	} else {
		if packetCredentials.APIKey, err = data.GetGlobalSecretKeySelectorValue(*spec.APIKeyReference); err != nil {
			return PacketCredentials{}, err
		}
	}

	if spec.ProjectID != "" {
		packetCredentials.ProjectID = spec.ProjectID
	} else {
		if packetCredentials.ProjectID, err = data.GetGlobalSecretKeySelectorValue(*spec.ProjectIDReference); err != nil {
			return PacketCredentials{}, err
		}
	}

	return packetCredentials, nil
}

func GetCredentials(data machinecontrollerData) (Credentials, error) {
	credentials := Credentials{}
	var err error

	if data.Cluster().Spec.Cloud.Packet != nil {
		if credentials.Packet, err = GetPacketCredentials(data); err != nil {
			return Credentials{}, err
		}
	}

	return credentials, err
}
