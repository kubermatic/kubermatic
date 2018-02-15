package instance

// Instance represents a instance on the cloud provider
type Instance interface {
	Name() string
	ID() string
	Addresses() []string
	Status() Status
}

type Status string

const (
	StatusRunning  Status = "running"
	StatusDeleting Status = "deleting"
	StatusDeleted  Status = "deleted"
	StatusCreating Status = "creating"
	StatusUnknown  Status = "unknown"
)
