package drives

import (
	"github.com/docker/machine/drivers/amazonec2"
	"github.com/docker/machine/drivers/digitalocean"
	"github.com/docker/machine/libmachine/drivers"
)

// Drivers holds all possible drivers.
// Each is for one Datacenter
var Drivers = map[string]drivers.Driver{
	"digitalocean": &digitalocean.Driver{},
	"aws":          &amazonec2.Driver{},
}
