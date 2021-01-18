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
	"fmt"
	"strings"
)

// AllExposeStrategies is a set containing all the ExposeStrategy.
var AllExposeStrategies = NewExposeStrategiesSet(ExposeStrategyNodePort, ExposeStrategyLoadBalancer, ExposeStrategyTunneling)

// ExposeStrategyFromString returns the expose strategy which String
// representation corresponds to the input string, and a bool saying whether a
// match was found or not.
func ExposeStrategyFromString(s string) (ExposeStrategy, bool) {
	es := ExposeStrategy(s)
	return es, AllExposeStrategies.Has(es)
}

// String returns the string representation of the ExposeStrategy.
func (e ExposeStrategy) String() string {
	return string(e)
}

// ExposeStrategiesSet is a set of ExposeStrategies.
type ExposeStrategiesSet map[ExposeStrategy]struct{}

// NewByte creates a ExposeStrategiesSet from a list of values.
func NewExposeStrategiesSet(items ...ExposeStrategy) ExposeStrategiesSet {
	es := ExposeStrategiesSet{}
	for _, item := range items {
		es[item] = struct{}{}
	}
	return es
}

// Has returns true if and only if item is contained in the set.
func (e ExposeStrategiesSet) Has(item ExposeStrategy) bool {
	_, contained := e[item]
	return contained
}

// Has returns true if and only if item is contained in the set.
func (e ExposeStrategiesSet) String() string {
	es := make([]string, 0, len(e))
	for k := range e {
		es = append(es, string(k))
	}
	// can be easily optimized in terms of allocations by using a bytes buffer
	// with some more verbosity, but this is not supposed to be called in a
	// perf critical path.
	return fmt.Sprintf("[%s]", strings.Join(es, ", "))
}
