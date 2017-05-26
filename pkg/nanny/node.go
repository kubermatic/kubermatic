package nanny

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"k8s.io/apimachinery/pkg/util/net"
)

// Node is the state of the actual node we're on
type Node struct {
	UID      string `json:"id"`
	CPUs     []*CPU `json:"cpus"`
	Memory   uint64 `json:"memory"`
	Space    uint64 `json:"space"`
	PublicIP string `json:"public_ip"`
}

// CPU is a CPU. Obviously. You're welcome.
type CPU struct {
	Cores     uint64  `json:"cores"`
	Frequency float64 `json:"frequency"`
}

// NewNodeFromSystemData creates a new node from system info received by psutil
func NewNodeFromSystemData(UID string) (n *Node, err error) {
	is, err := cpu.Info()
	if err != nil {
		return
	}

	var CPUs []*CPU
	for _, i := range is {
		CPUs = append(CPUs, &CPU{
			Frequency: i.Mhz,
			Cores:     uint64(i.Cores),
		})
	}

	m, err := mem.VirtualMemory()
	if err != nil {
		return
	}

	s, err := disk.Usage("/")
	if err != nil {
		return
	}

	ip, err := net.ChooseHostInterface()
	if err != nil {
		return
	}

	n = &Node{
		UID:      UID,
		CPUs:     CPUs,
		Memory:   m.Available,
		Space:    s.Free,
		PublicIP: ip.String(),
	}

	return
}
