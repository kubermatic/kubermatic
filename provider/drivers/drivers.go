package drivers

import (
	"encoding/json"
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

// auxDriverInstance for isolation.
type auxDriverInstance struct {
	Driver     `json:"driver"`
	DriverType string `json:"driver_type"`
}

func newAux(d *DriverInstance) *auxDriverInstance {
	return &auxDriverInstance{
		Driver:     d.Driver,
		DriverType: d.driverType,
	}
}

// DriverInstance is a instanciated Driver, which can be serialized over json.
type DriverInstance struct {
	Driver
	driverType string
}

// MarshalJSON marshales a struct into JSON.
func (d *DriverInstance) MarshalJSON() ([]byte, error) {
	return json.Marshal(newAux(d))
}

// UnmarshalJSON unmarshals from JSON.
func (d *DriverInstance) UnmarshalJSON(b []byte) error {
	var nameHolder struct {
		DriverType string `json:"driver_type"`
	}

	if err := json.Unmarshal(b, &nameHolder); err != nil {
		return err
	}

	// This creates a driver in the root dir but overwrites it
	// It can be higly dangerous when used with untrusted json.
	emptyDriverInstance, err := AvaliableDrivers.getEmptyDriver("", "")
	if err != nil {
		return err
	}
	aux := auxDriverInstance{Driver: emptyDriverInstance}

	if err = json.Unmarshal(b, &aux); err != nil {
		return err
	}

	*d = DriverInstance{
		Driver:     aux.Driver,
		driverType: aux.DriverType,
	}
	return nil
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

// CreateDriver creates a new Driver with a given name.
func (d Drivers) CreateDriver(drivername string, clustername string, flags flag.Flags) (*DriverInstance, error) {
	p, err := d.getEmptyDriver(drivername, clustername)
	if err != nil {
		return nil, err
	}
	p.SetConfigFromFlags(flags)
	return &DriverInstance{
		Driver:     p,
		driverType: drivername,
	}, nil
}

// GetEmptyDriver return a Driver with the underlying instanciated driver.
func (d Drivers) getEmptyDriver(drivername string, clustername string) (Driver, error) {
	nd, ok := d[drivername]
	if !ok {
		return nil, fmt.Errorf("Couldn't find the driver %q", drivername)
	}
	p := nd(clustername, clustername)
	return p, nil
}
