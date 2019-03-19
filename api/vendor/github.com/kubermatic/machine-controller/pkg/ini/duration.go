package ini

import (
	"time"
)

// Duration is the encoding.TextUnmarshaler interface for time.Duration
type Duration struct {
	time.Duration
}

// UnmarshalText is used to convert from text to Duration
func (d *Duration) UnmarshalText(text []byte) error {
	res, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	d.Duration = res
	return nil
}

// MarshalText is used to convert from Duration to text
func (d *Duration) MarshalText() []byte {
	return []byte(d.Duration.String())
}
