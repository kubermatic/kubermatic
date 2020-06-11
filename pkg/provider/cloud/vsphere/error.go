package vsphere

import (
	"github.com/vmware/govmomi/find"
)

func isNotFound(err error) bool {
	_, ok := err.(*find.NotFoundError)
	return ok
}
