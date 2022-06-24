/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package wait

import (
	"errors"
	"testing"
	"time"

	k8swait "k8s.io/apimachinery/pkg/util/wait"
)

func TestPollSuccess(t *testing.T) {
	executions := 0

	err := Poll(1*time.Millisecond, 100*time.Millisecond, func() (error, error) {
		executions++
		return nil, nil
	})

	if err != nil {
		t.Fatalf("Poll should have returned nil, but returned %v", err)
	}

	if executions > 1 {
		t.Fatalf("Poll should have only executed the condition once, but ran it %d times", executions)
	}
}

func TestPollTimeout(t *testing.T) {
	err := Poll(1*time.Millisecond, 10*time.Millisecond, func() (error, error) {
		return errors.New("transient"), nil
	})

	if err == nil {
		t.Fatal("Poll should have returned an error, but got nil")
	}

	if !errors.Is(err, k8swait.ErrWaitTimeout) {
		t.Fatalf("err should be a wrapped ErrWaitTimeout, but is %+v", err)
	}
}

func TestPollTerminalError(t *testing.T) {
	executions := 0
	terminal := errors.New("terminal")

	err := Poll(1*time.Millisecond, 10*time.Millisecond, func() (error, error) {
		executions++
		return nil, terminal
	})

	if err == nil {
		t.Fatal("Poll should have returned an error, but got nil")
	}

	// This specifically ensures that we get exactly the error that is returned
	// by the condition, without any kind of wrapping.

	//nolint:errorlint
	if err != terminal {
		t.Fatalf("err should has been the terminal error, but is %+v", err)
	}
}
