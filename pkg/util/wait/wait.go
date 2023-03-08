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
	"time"

	"go.uber.org/zap"

	k8swait "k8s.io/apimachinery/pkg/util/wait"
)

type ConditionFunc func() (transient error, terminal error)
type PollFunc func(interval, timeout time.Duration, condition k8swait.ConditionFunc) error

// Poll works identically to k8swait.Poll, with the exception that a condition
// must return an error/nil to indicate a successful condition. In case a timeout
// occurs, the transient error is returned as part of the k8swait.ErrWaitTimeout,
// but note that the ErrWaitTimeout is being wrapped and the transient error only
// included as a string.
func Poll(ctx context.Context, interval, timeout time.Duration, condition ConditionFunc) error {
	return enrich(k8swait.Poll, ctx, nil, interval, timeout, condition)
}

// PollLog is an extension of Poll and will, if a transient error occurs,
// log that error on the INFO level using the given logger. Use this if you
// want continuous feedback and make sure to set a sensible interval
// like 5+ seconds.
func PollLog(ctx context.Context, log *zap.SugaredLogger, interval, timeout time.Duration, condition ConditionFunc) error {
	return enrich(k8swait.Poll, ctx, log, interval, timeout, condition)
}

// PollImmediate works identically to k8swait.PollImmediate, with the exception
// that a condition must return an error/nil to indicate a successful condition.
// In case a timeout occurs, the transient error is returned as part of the
// k8swait.ErrWaitTimeout, but note that the ErrWaitTimeout is being wrapped and
// the transient error only included as a string.
func PollImmediate(ctx context.Context, interval, timeout time.Duration, condition ConditionFunc) error {
	return enrich(k8swait.PollImmediate, ctx, nil, interval, timeout, condition)
}

// PollImmediateLog is an extension of PollImmediate and will, if a transient
// error occurs, log that error on the INFO level using the given logger.
// Use this if you want continuous feedback and make sure to set a sensible interval
// like 5+ seconds.
func PollImmediateLog(ctx context.Context, log *zap.SugaredLogger, interval, timeout time.Duration, condition ConditionFunc) error {
	return enrich(k8swait.PollImmediate, ctx, log, interval, timeout, condition)
}

func enrich(poller PollFunc, ctx context.Context, log *zap.SugaredLogger, interval, timeout time.Duration, condition ConditionFunc) error {
	var lastErr error

	waitErr := poller(interval, timeout, func() (done bool, err error) {
		// stop waiting if the given context was cancelled or timed out
		if err := ctx.Err(); err != nil {
			return false, err
		}

		transient, terminal := condition()
		if terminal != nil {
			return false, terminal
		}

		lastErr = transient

		// If a logger is given, we provide continuous feedback about the condition.
		if transient != nil && log != nil {
			log.Infof("Waiting: %s", transient.Error())
		}

		return transient == nil, nil
	})

	if errors.Is(waitErr, k8swait.ErrWaitTimeout) && lastErr != nil {
		waitErr = fmt.Errorf("%w; last error was: %w", waitErr, lastErr)
	}

	return waitErr
}
