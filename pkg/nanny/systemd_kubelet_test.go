package nanny

import (
	"io/ioutil"
	"net"
	"os"
	"path"
	"testing"

	"github.com/coreos/go-systemd/dbus"
)

type testConnection struct {
	unitStatus []dbus.UnitStatus
	reloaded   bool
	running    bool
}

func (c *testConnection) Reload() error {
	c.reloaded = true
	return nil
}

func (c *testConnection) StartUnit(name string, mode string, ch chan<- string) (int, error) {
	c.running = true
	return 1, nil
}

func (c *testConnection) StopUnit(name string, mode string, ch chan<- string) (int, error) {
	c.running = false
	return 1, nil
}

func (c *testConnection) ListUnitsByNames(units []string) ([]dbus.UnitStatus, error) {
	return c.unitStatus, nil
}

var conn *testConnection

func newTestInterface() KubeletInterface {
	conn = &testConnection{}

	return &SystemdKubelet{
		conn: conn,
	}
}

func TestNewSystemdKubelet(t *testing.T) {
	t.Skip("Not working yet since creating /run is not allowed")

	err := os.MkdirAll(path.Dir("/run/systemd/private"), 0777)
	if err != nil {
		t.Fatal(err)
	}

	l, err := net.Listen("unix", "unix:path=/run/systemd/private")
	if err != nil {
		t.Fatalf("Error opening systemd socket: %v", err)
	}

	defer func(l net.Listener) {
		err := l.Close()
		if err != nil {
			t.Fatalf("Error closing listener socket: %v", err)
		}
	}(l)
	go func(l net.Listener) {
		for {
			_, err := l.Accept()
			if err != nil {
				return
			}
		}
	}(l)

	k, err := NewSystemdKubelet()
	if err != nil {
		t.Fatalf("Error calling NewSystemdKubelet: %v", err)
	}

	if k == nil {
		t.Fatal("Expected NewSystemdKubelet to return a kubelet instance, got nil")
	}
}

func TestSystemdKubelet_WriteKubeConfig(t *testing.T) {
	k := newTestInterface()
	c := &Cluster{KubeConfig: "asdf"}
	n := &Node{UID: "qwertz"}
	f := path.Join(os.TempDir(), "kubeconfig.test.tmp")
	defer deleteTempFile(t, f)

	if err := k.WriteKubeConfig(n, c, f); err != nil {
		t.Fatal(err)
	}

	if b, err := ioutil.ReadFile(f); err != nil {
		t.Fatal(err)
	} else if string(b) != c.KubeConfig {
		t.Fatalf("Expected kubeconfig file to contain %q, got %q", c.KubeConfig, string(b))
	}
}

var kubeletTemplateResult = `[Unit]
Description=Kubernetes Kubelet

[Service]
Restart=always
RestartSec=10
Environment="PATH=/opt/bin:/usr/bin:/usr/sbin:$PATH"
ExecStartPre=/usr/bin/mkdir -p /var/lib/kubelet /etc/kubernetes/manifests
ExecStartPre=/usr/bin/curl -L -o /var/lib/kubelet/kubelet https://storage.googleapis.com/kubernetes-release/release/v1.5.4/bin/linux/amd64/kubelet
ExecStartPre=/usr/bin/chmod +x /var/lib/kubelet/kubelet
ExecStartPre=/usr/bin/mkdir -p /opt/bin
ExecStartPre=/usr/bin/curl -L -o /opt/bin/socat https://s3-eu-west-1.amazonaws.com/kubermatic/coreos/socat
ExecStartPre=/usr/bin/chmod +x /opt/bin/socat
ExecStart=/var/lib/kubelet/kubelet \
  --address=0.0.0.0 \
  --kubeconfig=/var/run/kubelet/kubeconfig \
  --require-kubeconfig \
  --cluster-dns=10.10.10.10 \
  --cluster-domain=cluster.local \
  --allow-privileged=true \
  --hostname-override="qwertz" \
  --network-plugin=cni

[Install]
WantedBy=multi-user.target`

func TestSystemdKubelet_WriteStartConfig(t *testing.T) {
	k := newTestInterface()
	c := &Cluster{KubeConfig: "asdf", APIServerURL: "http://abc.de:1234"}
	n := &Node{UID: "qwertz"}
	f := path.Join(os.TempDir(), "kubeconfig.test.tmp")
	defer deleteTempFile(t, f)

	if err := k.WriteStartConfig(n, c, f); err != nil {
		t.Fatal(err)
	}

	if b, err := ioutil.ReadFile(f); err != nil {
		t.Fatal(err)
	} else if string(b) != kubeletTemplateResult {
		t.Fatalf("Expected kubeconfig file to contain %q, got %q", kubeletTemplateResult, string(b))
	}
}

func TestSystemdKubelet_Running(t *testing.T) {
	k := newTestInterface()
	c := &Cluster{}

	conn.unitStatus = append(conn.unitStatus, dbus.UnitStatus{
		ActiveState: "active",
		LoadState:   "loaded",
	})

	r, err := k.Running(c)
	if err != nil {
		t.Fatalf("Expected Running to finish wiothout returning an error, got %v", err)
	}

	if r != true {
		t.Errorf("Expected Running to return true, got %t", r)
	}
}

func TestSystemdKubelet_RunningWrongStateLoaded(t *testing.T) {
	k := newTestInterface()
	c := &Cluster{}

	conn.unitStatus = append(conn.unitStatus, dbus.UnitStatus{
		ActiveState: "active",
		LoadState:   "not_loaded",
	})

	r, err := k.Running(c)
	if err != nil {
		t.Fatalf("Expected Running to finish wiothout returning an error, got %v", err)
	}

	if r != false {
		t.Errorf("Expected Running to return false, got %t", r)
	}
}

func TestSystemdKubelet_RunningWrongStateActive(t *testing.T) {
	k := newTestInterface()
	c := &Cluster{}

	conn.unitStatus = append(conn.unitStatus, dbus.UnitStatus{
		ActiveState: "not_active",
		LoadState:   "loaded",
	})

	r, err := k.Running(c)
	if err != nil {
		t.Fatalf("Expected Running to finish wiothout returning an error, got %v", err)
	}

	if r != false {
		t.Errorf("Expected Running to return false, got %t", r)
	}
}

func TestSystemdKubelet_Start(t *testing.T) {
	k := newTestInterface()
	c := &Cluster{}

	err := k.Start(c)
	if err != nil {
		t.Fatalf("Expected Running to finish wiothout returning an error, got %v", err)
	}

	if conn.running != true {
		t.Error("Expected connection to be in running state, but isn't.")
	}

	if conn.reloaded != true {
		t.Error("Expected connection to be reloaded, but isn't.")
	}
}

func TestSystemdKubelet_Stop(t *testing.T) {
	k := newTestInterface()
	c := &Cluster{}
	conn.running = true

	err := k.Stop(c)
	if err != nil {
		t.Fatalf("Expected Running to finish wiothout returning an error, got %v", err)
	}

	if conn.running != false {
		t.Error("Expected connection to be in running state, but isn't.")
	}
}

func deleteTempFile(t *testing.T, name string) {
	// When the file doesn't exist, do nothing
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return
		}

		t.Fatal(err)
	}

	err := os.Remove(name)
	if err != nil {
		t.Fatal(err)
	}
}
