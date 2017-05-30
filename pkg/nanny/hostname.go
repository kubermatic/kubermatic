package nanny

import (
	"fmt"
	"os"
)

const (
	hostsFile    = "/etc/hosts"
	nodeNameFile = "/opt/node-name"
)

// AddAsLocalHostname adds the uid to the hosts file with 127.0.0.1 as address
func AddAsLocalHostname(uid string) error {
	f, err := os.OpenFile(hostsFile, os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open hosts file %q: %v", hostsFile, err)
	}

	_, err = f.WriteString(fmt.Sprintf("127.0.0.1	%s\n", uid))
	if err != nil {
		return fmt.Errorf("failed to add %s as hostname for 127.0.0.1: %v", uid, err)
	}
	return nil
}

// WriteNodeName writes the name of the node to /opt/node-name
func WriteNodeName(uid string) error {
	f, err := os.OpenFile(nodeNameFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open node name file %q: %v", nodeNameFile, err)
	}

	_, err = f.WriteString(uid)
	if err != nil {
		return fmt.Errorf("failed to write %q to node name file %q: %v", uid, nodeNameFile, err)
	}
	return nil
}
