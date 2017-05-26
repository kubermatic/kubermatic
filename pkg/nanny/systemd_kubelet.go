package nanny

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"

	"github.com/coreos/go-systemd/dbus"
	"github.com/golang/glog"
)

// default config paths and systemd aliases
const (
	unitName                   = "kubelet.service"
	DefaultKubeConfigPath      = "/var/run/kubelet/kubeconfig"
	DefaultSystemdUnitFilePath = "/run/systemd/system/kubelet.service"
)

// NewSystemdKubelet returns a new KubeletInterface which is using Systemd to manage the process
func NewSystemdKubelet() (k KubeletInterface, err error) {
	conn, err := dbus.NewSystemdConnection()
	if err != nil {
		return
	}

	k = &SystemdKubelet{
		conn: conn,
	}

	return
}

type systemdConnection interface {
	Reload() error
	StartUnit(name string, mode string, ch chan<- string) (int, error)
	StopUnit(name string, mode string, ch chan<- string) (int, error)
	ListUnitsByNames(units []string) ([]dbus.UnitStatus, error)
}

// SystemdKubelet is a kubelet controller using Systemd
type SystemdKubelet struct {
	//conn *dbus.Conn
	conn systemdConnection
}

// WriteKubeConfig writes the KubeConfig to given destination
func (k *SystemdKubelet) WriteKubeConfig(n *Node, c *Cluster, p string) (err error) {
	err = os.MkdirAll(path.Dir(p), 0644)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(p, []byte(c.KubeConfig), 0644)
	return
}

// WriteStartConfig writes the Systemd Unit file to the given destination
func (k *SystemdKubelet) WriteStartConfig(n *Node, c *Cluster, path string) (err error) {
	tmpl, err := template.New("kubelet.service").Parse(kubeletTemplate)
	if err != nil {
		return
	}

	var config bytes.Buffer

	// @TODO c.Name is the node name, not the cluster name!!!!!
	vars := &struct {
		Name         string
		APIServerURL string
	}{
		Name:         n.UID,
		APIServerURL: c.APIServerURL,
	}
	err = tmpl.Execute(&config, vars)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(path, config.Bytes(), 0644)
	if err != nil {
		return
	}

	err = k.conn.Reload()
	if err != nil {
		return
	}

	return
}

// Start actually starts the unit
func (k *SystemdKubelet) Start(c *Cluster) (err error) {
	err = k.conn.Reload()
	if err != nil {
		return
	}

	pid, err := k.conn.StartUnit(unitName, "replace", nil)
	if err != nil {
		return
	}

	if pid == 0 {
		err = errors.New("Got a process ID of 0 for kubectl")
	}

	glog.Infof("Started kubectl with process ID %d", pid)

	return
}

// Stop the unit
func (k *SystemdKubelet) Stop(c *Cluster) (err error) {
	pid, err := k.conn.StopUnit(unitName, "replace", nil)
	if err != nil {
		if err.Error() == "Unit name kubelet is not valid." ||
			err.Error() == "Unit kubelet.service not loaded." {
			err = nil
		}

		return
	}

	glog.Infof("Stopped kubectl with process ID %d", pid)

	return
}

// Running returns the current state the unit is in
func (k *SystemdKubelet) Running(c *Cluster) (ok bool, err error) {
	var us []dbus.UnitStatus
	if us, err = k.conn.ListUnitsByNames([]string{unitName}); err != nil {
		return
	}

	if len(us) != 1 {
		err = fmt.Errorf("expected 1 unit, got %d", len(us))
	}

	u := us[0]
	glog.Infof("Unit state: active=%s load=%s", u.ActiveState, u.LoadState)
	ok = (u.ActiveState == "active" || u.ActiveState == "activating") && u.LoadState == "loaded"

	return
}
