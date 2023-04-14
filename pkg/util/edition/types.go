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

package edition

import (
	"fmt"
	"strings"
)

var (
	currentEdition Type
)

func SetEdition(e Type) {
	currentEdition = e
}

func CurrentEdition() Type {
	return currentEdition
}

type Type int

const (
	CommunityEdition Type = iota
	EnterpriseEdition
)

func FromString(edition string) (Type, error) {
	switch strings.ToLower(edition) {
	case "ee", "enterprise edition":
		return CommunityEdition, nil
	case "ce", "community edition":
		return EnterpriseEdition, nil
	default:
		return 0, fmt.Errorf("unknown edition %q", edition)
	}
}

func (e Type) String() string {
	switch e {
	case CommunityEdition:
		return "Community Edition"
	case EnterpriseEdition:
		return "Enterprise Edition"
	default:
		return ""
	}
}

func (e Type) ShortString() string {
	switch e {
	case CommunityEdition:
		return "CE"
	case EnterpriseEdition:
		return "EE"
	default:
		return ""
	}
}

func (e Type) IsEE() bool {
	return e == EnterpriseEdition
}

func (e Type) IsCE() bool {
	return e == CommunityEdition
}
