package controller

// Controller represents an control loop started with Run and terminated
// by closing the stopCh.
type Controller interface {
	// Run start the control loop, which terminates when stopCh is closed.
	Run(stopCh <-chan struct{})
}
