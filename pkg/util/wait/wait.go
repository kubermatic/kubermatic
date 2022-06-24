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
	"fmt"
	"time"

	"go.uber.org/zap"

	k8swait "k8s.io/apimachinery/pkg/util/wait"
)

type ConditionFunc func() (transient error, terminal error)
type PollFunc func(interval, timeout time.Duration, condition k8swait.ConditionFunc) error

func Poll(interval, timeout time.Duration, condition ConditionFunc) error {
	return enrich(k8swait.Poll, nil, interval, timeout, condition)
}

func PollLog(log *zap.SugaredLogger, interval, timeout time.Duration, condition ConditionFunc) error {
	return enrich(k8swait.Poll, log, interval, timeout, condition)
}

func PollImmediate(interval, timeout time.Duration, condition ConditionFunc) error {
	return enrich(k8swait.PollImmediate, nil, interval, timeout, condition)
}

func PollImmediateLog(log *zap.SugaredLogger, interval, timeout time.Duration, condition ConditionFunc) error {
	return enrich(k8swait.PollImmediate, log, interval, timeout, condition)
}

func enrich(upstream PollFunc, log *zap.SugaredLogger, interval, timeout time.Duration, condition ConditionFunc) error {
	var lastErr error

	waitErr := upstream(interval, timeout, func() (done bool, err error) {
		transient, terminal := condition()
		if terminal != nil {
			return false, terminal
		}

		lastErr = transient

		// If a logger is given, we provide continuous feedback about the condition.
		if transient != nil && log != nil {
			log.Infow("Waiting", "status", transient.Error())
		}

		return transient == nil, nil
	})

	if errors.Is(waitErr, k8swait.ErrWaitTimeout) && lastErr != nil {
		waitErr = fmt.Errorf("%w; last error was: %v", waitErr, lastErr)
	}

	return waitErr
}
