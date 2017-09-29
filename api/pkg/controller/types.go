package controller

// Interface represents an control loop started with Run and terminated
// by closing the stopCh.
type Interface interface {
	// Run start the control loop, which terminates when stopCh is closed.
	Run(workerCount int, stopCh chan struct{})
}
