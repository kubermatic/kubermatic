package nanny

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path"
	"strconv"

	"github.com/golang/glog"
)

const (
	sshDirName             = ".ssh"
	authorizedKeysFilename = "authorized_keys"
	authorizedKeysPerm     = 0600
)

// SSHPubKeyManager is a interface for ssh pub key management
type SSHPubKeyManager interface {
	Flush() error
	AddPubKey(key string, flush bool) ([]string, error)
}

// NewUserSSHKeyManager returns a file based ssh pub key manager
func NewUserSSHKeyManager(username string) (SSHPubKeyManager, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user %q", username)
	}
	sshConfigPath := path.Join(u.HomeDir, sshDirName)
	_, err = os.Stat(sshConfigPath)
	if os.IsNotExist(err) {
		err = os.Mkdir(sshConfigPath, 0777)
		if err != nil {
			return nil, fmt.Errorf("failed to create ssh config dir %q for %q: %v", sshConfigPath, username, err)
		}
	}

	authKeysPath := path.Join(u.HomeDir, sshDirName, authorizedKeysFilename)
	_, err = os.Stat(authKeysPath)
	if os.IsNotExist(err) {
		f, err := os.Create(authKeysPath)
		defer func() {
			err := f.Close()
			if err != nil {
				glog.Error(err)
			}
		}()
		if err != nil {
			return nil, fmt.Errorf("failed to create %q: %v", authKeysPath, err)
		}

		uid, err := strconv.Atoi(u.Uid)
		if err != nil {
			return nil, fmt.Errorf("failed to convert uid %q to int: %v", u.Uid, err)
		}
		gid, err := strconv.Atoi(u.Gid)
		if err != nil {
			return nil, fmt.Errorf("failed to convert gid %q to int: %v", u.Gid, err)
		}

		err = os.Chown(authKeysPath, uid, gid)
		if err != nil {
			return nil, fmt.Errorf("failed to chown %q: %v", authKeysPath, err)
		}

		err = os.Chmod(authKeysPath, authorizedKeysPerm)
		if err != nil {
			return nil, fmt.Errorf("failed to chmod %q: %v", authKeysPath, err)
		}
	}

	return &SSHPubKeyFileWriter{
		u: u,
	}, nil
}

// SSHPubKeyFileWriter is a file based ssh pub key manager
type SSHPubKeyFileWriter struct {
	u *user.User
}

// Flush truncates the authorized_keys file
func (fw *SSHPubKeyFileWriter) Flush() error {
	authKeysPath := path.Join(fw.u.HomeDir, sshDirName, authorizedKeysFilename)
	_, err := os.Stat(authKeysPath)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(authKeysPath, os.O_TRUNC|os.O_WRONLY, authorizedKeysPerm)
	if err != nil {
		return err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			glog.Error(err)
		}
	}()
	_, err = f.WriteString("")
	return err
}

// AddPubKey adds a public ssh key. flush would truncate the authorized_keys file before adding
func (fw *SSHPubKeyFileWriter) AddPubKey(key string, flush bool) ([]string, error) {
	var err error
	keys := []string{}
	if flush {
		err = fw.Flush()
		if err != nil {
			return keys, fmt.Errorf("could not flush before adding keys: %v", err)
		}
	}

	authKeysPath := path.Join(fw.u.HomeDir, sshDirName, authorizedKeysFilename)
	f, err := os.OpenFile(authKeysPath, os.O_APPEND|os.O_RDWR, authorizedKeysPerm)
	if err != nil {
		return keys, err
	}

	_, err = f.WriteString(key)
	if err != nil {
		return keys, fmt.Errorf("failed to add key %q to %q: %v", key, authKeysPath, err)
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		keys = append(keys, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return keys, fmt.Errorf("failed reading lines from %q: %v", authKeysPath, err)
	}
	return keys, nil
}
