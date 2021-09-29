package utils

import "time"

const (
	ClusterReadinessCheckPeriod = 10 * time.Second
	ClusterReadinessTimeout     = 10 * time.Minute

	MachineDeploymentName      = "ccm-migration-e2e"
	MachineDeploymentNamespace = "kube-system"

	KubeletVersion = "1.20.0"
)
