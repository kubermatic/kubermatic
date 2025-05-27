/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"encoding/json"
	"time"
)

// Time is a wrapper around time.Time which supports correct
// marshaling JSON.  Wrappers are provided for many
// of the factory methods that the time package offers.
type Time struct {
	time.Time `json:"time,omitempty"`
}

// String returns the representation of the time.
func (t *Time) String() string {
	return t.Time.String()
}

// NewTime returns a wrapped instance of the provided time.
func NewTime(t time.Time) Time {
	return Time{t}
}

// Date returns the Time corresponding to the supplied parameters
// by wrapping time.Date.
func Date(year int, month time.Month, day, hour, minute, sec, nsec int, loc *time.Location) Time {
	return Time{time.Date(year, month, day, hour, minute, sec, nsec, loc)}
}

// Now returns the current local time.
func Now() Time {
	return Time{time.Now()}
}

// MarshalJSON implements the json.Marshaler interface.
func (t *Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.UTC().Format(time.RFC3339))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *Time) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && string(b) == "\"0001-01-01T00:00:00Z\"" {
		t.Time = time.Time{}
		return nil
	}

	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	pt, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}

	t.Time = pt.Local()
	return nil
}

// Before reports whether the time instant t is before u.
func (t *Time) Before(u Time) bool {
	return t.Time.Before(u.Time)
}

// Equal reports whether the time instant t is equal to u.
func (t *Time) Equal(u *Time) bool {
	if t == nil && u == nil {
		return true
	}
	if t != nil && u != nil {
		return t.Time.Equal(u.Time)
	}
	return false
}

// IsZero returns true if the value is nil or time is zero.
func (t *Time) IsZero() bool {
	if t == nil {
		return true
	}
	return t.Time.IsZero()
}
