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

package semver

import (
	"flag"
	"fmt"

	semverlib "github.com/Masterminds/semver/v3"
)

var (
	_ flag.Value = new(Semver)
)

// Semver is a type that encapsulates github.com/Masterminds/semver/v3.Version struct so it can be used in our API.
type Semver string

// NewSemver creates new Semver version struct and returns pointer to it.
func NewSemver(ver string) (*Semver, error) {
	v := new(Semver)
	if err := v.Set(ver); err != nil {
		return nil, err
	}

	return v, nil
}

// NewSemverOrDie behaves similar to NewVersion, i.e. it creates new Semver version struct, but panics if an error happens.
func NewSemverOrDie(ver string) *Semver {
	sv, err := NewSemver(ver)
	if err != nil {
		panic(err)
	}

	return sv
}

// Set initializes semver struct and sets version.
func (s *Semver) Set(ver string) error {
	if _, err := semverlib.NewVersion(ver); err != nil {
		return err
	}
	*s = Semver(ver)

	return nil
}

// Semver returns github.com/Masterminds/semver/v3 struct.
// In case when Semver is nil, nil will be returned.
// In case of parsing error, nil will be returned.
func (s *Semver) Semver() *semverlib.Version {
	if s == nil {
		return nil
	}

	sver, err := semverlib.NewVersion(string(*s))
	if err != nil {
		return nil
	}

	return sver
}

// Equal compares two version structs by comparing Semver values.
func (s *Semver) Equal(b *Semver) bool {
	if s == nil || b == nil {
		return false
	}

	sver, bver := s.Semver(), b.Semver()
	if sver == nil || bver == nil {
		return false
	}

	return sver.Equal(bver)
}

func (s *Semver) LessThan(b *Semver) bool {
	if s == nil || b == nil {
		return false
	}

	sver, bver := s.Semver(), b.Semver()
	if sver == nil || bver == nil {
		return false
	}

	return sver.LessThan(bver)
}

func (s *Semver) GreaterThan(b *Semver) bool {
	if s == nil || b == nil {
		return false
	}

	sver, bver := s.Semver(), b.Semver()
	if sver == nil || bver == nil {
		return false
	}

	return sver.GreaterThan(bver)
}

// String returns string representation of Semver version.
func (s *Semver) String() string {
	sver := s.Semver()
	if sver == nil {
		return ""
	}

	return sver.String()
}

// MajorMinor returns a string like "Major.Minor".
func (s *Semver) MajorMinor() string {
	sver := s.Semver()
	if sver == nil {
		return ""
	}

	return fmt.Sprintf("%d.%d", sver.Major(), sver.Minor())
}

func (s Semver) DeepCopy() Semver {
	if s.Semver() == nil {
		return ""
	}

	return *NewSemverOrDie(s.String())
}

func (s *Semver) DeepCopyInto(out *Semver) {
	*out = s.DeepCopy()
}
