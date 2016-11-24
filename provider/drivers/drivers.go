package drivers

import (
	"fmt"

	"github.com/docker/machine/drivers/amazonec2"
	"github.com/docker/machine/drivers/digitalocean"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/kubermatic/api/provider/drivers/flag"
)

// Driver is a docker libmachine driver
type Driver interface {
	drivers.Driver
}

// DriverFactory is used to create a Driver.
type DriverFactory func(name, path string) Driver

// Drivers holds all possible drivers.
// Each is for one Datacenter
type Drivers map[string]DriverFactory

// AvaliableDrivers contains all drivers that are avaliable to use.
var AvaliableDrivers = Drivers{
	"digitalocean": func(name, path string) Driver {
		d := digitalocean.NewDriver(name, path)
		return d
	},
	"aws": func(name, path string) Driver {
		d := amazonec2.NewDriver(name, path)
		return d
	},
}

// CreateHost creates a new Driver with a given name.
func (d Drivers) CreateHost(drivername string, clustername string, flags flag.Flags) (Driver, error) {
	nd, ok := d[drivername]
	if !ok {
		return nil, fmt.Errorf("Couldn't find the driver %q", drivername)
	}
	p := nd(clustername, clustername)
	p.SetConfigFromFlags(flags)
	return p, nil
}
