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
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPollSuccess(t *testing.T) {
	executions := 0

	err := Poll(context.Background(), 1*time.Millisecond, 100*time.Millisecond, func(ctx context.Context) (error, error) {
		executions++
		return nil, nil
	})

	if err != nil {
		t.Fatalf("Poll should have returned nil, but returned %v", err)
	}

	if executions != 1 {
		t.Fatalf("Poll should have only executed the condition once, but ran it %d times", executions)
	}
}

func TestPollTimeout(t *testing.T) {
	err := Poll(context.Background(), 1*time.Millisecond, 10*time.Millisecond, func(ctx context.Context) (error, error) {
		return errors.New("transient"), nil
	})

	if err == nil {
		t.Fatal("Poll should have returned an error, but got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err should be a wrapped DeadlineExceeded, but is %+v", err)
	}

	if !strings.Contains(err.Error(), "transient") {
		t.Fatalf("err should have returned the transient error message, but was: %v", err)
	}
}

func TestPollTimeoutKeepsErrorFromBeforeDeadline(t *testing.T) {
	executions := 0

	err := Poll(context.Background(), 1*time.Millisecond, 50*time.Millisecond, func(ctx context.Context) (error, error) {
		executions++
		if executions == 1 {
			return errors.New("2 of 3 nodes are ready"), nil
		}

		// simulate an API call that is still running when the timeout hits
		<-ctx.Done()
		return fmt.Errorf("client rate limiter Wait returned an error: %w", ctx.Err()), nil
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err should be a wrapped DeadlineExceeded, but is %+v", err)
	}

	if !strings.Contains(err.Error(), "2 of 3 nodes are ready") {
		t.Fatalf("err should contain the last error from before the deadline, but was: %v", err)
	}

	if strings.Contains(err.Error(), "rate limiter") {
		t.Fatalf("err should not contain the error caused by the expired context, but was: %v", err)
	}
}

func TestPollTimeoutDuringFirstAttempt(t *testing.T) {
	err := Poll(context.Background(), 1*time.Millisecond, 50*time.Millisecond, func(ctx context.Context) (error, error) {
		<-ctx.Done()
		return fmt.Errorf("client rate limiter Wait returned an error: %w", ctx.Err()), nil
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err should be a wrapped DeadlineExceeded, but is %+v", err)
	}

	if !strings.Contains(err.Error(), "rate limiter") {
		t.Fatalf("err should fall back to the only error the condition returned, but was: %v", err)
	}
}

func TestPollTerminalError(t *testing.T) {
	executions := 0
	terminal := errors.New("terminal")

	err := Poll(context.Background(), 1*time.Millisecond, 10*time.Millisecond, func(ctx context.Context) (error, error) {
		executions++
		return nil, terminal
	})

	if err == nil {
		t.Fatal("Poll should have returned an error, but got nil")
	}

	if executions != 1 {
		t.Fatalf("Poll should have only executed the condition once, but ran it %d times", executions)
	}

	// This specifically ensures that we get exactly the error that is returned
	// by the condition, without any kind of wrapping.

	//nolint:errorlint
	if err != terminal {
		t.Fatalf("err should has been the terminal error, but is %+v", err)
	}
}
