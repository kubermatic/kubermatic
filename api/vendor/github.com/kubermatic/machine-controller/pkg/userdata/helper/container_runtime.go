package helper

import (
	"bytes"
	"fmt"
	"text/template"
)

// DockerDaemonConfig returns the docker daemon.json with preferred settings
func DockerDaemonConfig() string {
	// We need to specify a custom default runtime, as docker does not allow to specify a path for runc (Which is a reserved key - which cannot be overwritten)
	// This is needed to make sure we only use the binaries we downloaded. On CoreOS the same binaries already exist on the OS, but in different versions.
	// We must only use our binaries to avoid versions conflicts
	return `{
  "default-runtime": "docker-runc",
  "runtimes": {
    "docker-runc": {
      "path": "/opt/bin/docker-runc"
    }
  },
  "storage-driver": "overlay2",
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "2"
  }
}`
}

const dockerSystemdUnitTpl = `[Unit]
Description=Docker Application Container Engine
Documentation=https://docs.docker.com
After=network-online.target
Wants=network-online.target

[Service]
Environment="PATH=/opt/bin:/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin/"
Type=notify
# the default is not to use systemd for cgroups because the delegate issues still
# exists and systemd currently does not support the cgroup feature set required
# for containers run by docker
ExecStart=/opt/bin/dockerd
ExecReload=/bin/kill -s HUP $MAINPID
LimitNOFILE=1048576
# Having non-zero Limit*s causes performance problems due to accounting overhead
# in the kernel. We recommend using cgroups to do container-local accounting.
LimitNPROC=infinity
LimitCORE=infinity
# Uncomment TasksMax if your systemd version supports it.
# Only systemd 226 and above support this version.
{{ if .SetTasksMax }}
TasksMax=infinity
{{ end }}
TimeoutStartSec=0
# set delegate yes so that systemd does not reset the cgroups of docker containers
Delegate=yes
# kill only the docker process, not all processes in the cgroup
KillMode=process
# restart the docker process if it exits prematurely
Restart=on-failure
StartLimitBurst=3
StartLimitInterval=60s

[Install]
WantedBy=multi-user.target`

// DockerSystemdUnit returns the systemd unit for docker. setTasksMax should be set if the consumer uses systemd > 226 (Ubuntu & CoreoS - NOT CentOS)
func DockerSystemdUnit(setTasksMax bool) (string, error) {
	tmpl, err := template.New("docker-systemd-unit").Funcs(TxtFuncMap()).Parse(dockerSystemdUnitTpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse docker-systemd-unit template: %v", err)
	}

	data := struct {
		SetTasksMax bool
	}{
		SetTasksMax: setTasksMax,
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute docker-systemd-unit template: %v", err)
	}

	return b.String(), nil
}

const (
	containerdSystemdUnit = `[Unit]
Description=containerd container runtime
Documentation=https://containerd.io
After=network.target

[Service]
Environment="PATH=/opt/bin:/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin/"
Delegate=yes
ExecStart=/opt/bin/docker-containerd -l unix:///var/run/docker/libcontainerd/docker-containerd.sock --metrics-interval=0 --start-timeout 2m --state-dir /var/run/docker/libcontainerd/containerd --shim /opt/bin/docker-containerd-shim --runtime /opt/bin/docker-runc

KillMode=process
Restart=always

# (lack of) limits from the upstream docker service unit
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
{{ if .SetTasksMax }}
TasksMax=infinity
{{ end }}

[Install]
WantedBy=multi-user.target`
)

// ContainerdSystemdUnit returns the systemd unit for containerd
func ContainerdSystemdUnit(setTasksMax bool) (string, error) {
	tmpl, err := template.New("containerd-systemd-unit").Funcs(TxtFuncMap()).Parse(containerdSystemdUnit)
	if err != nil {
		return "", fmt.Errorf("failed to parse containerd-systemd-unit template: %v", err)
	}

	data := struct {
		SetTasksMax bool
	}{
		SetTasksMax: setTasksMax,
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute containerd-systemd-unit template: %v", err)
	}

	return b.String(), nil
}
