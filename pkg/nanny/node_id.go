package nanny

import (
	"io/ioutil"
	"strings"
)

// DefaultNodeIDFile is the default location of the node ID
const DefaultNodeIDFile string = "/proc/sys/kernel/random/boot_id"

// LoadNodeID tries to load the content of the given file as a node ID string
func LoadNodeID(path string) (string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	UID := strings.TrimSpace(string(b))
	return UID, nil
}
