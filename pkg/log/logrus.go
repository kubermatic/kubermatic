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

package log

import (
	"context"

	"github.com/sirupsen/logrus"
)

func NewLogrus() *logrus.Logger {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
	}

	logger.AddHook(&prefixHook{})

	return logger
}

type prefixKeyType string

const prefixKey prefixKeyType = "prefix"

type prefixHook struct{}

func (h *prefixHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *prefixHook) Fire(e *logrus.Entry) error {
	if e.Context != nil {
		prefix := e.Context.Value(prefixKey)
		if prefix != nil {
			e.Message = prefix.(string) + e.Message
		}
	}

	return nil
}

func Prefix(e *logrus.Entry, prefix string) *logrus.Entry {
	parentCtx := e.Context
	if parentCtx == nil {
		parentCtx = context.Background()
	}

	if oldPrefix := parentCtx.Value(prefixKey); oldPrefix != nil {
		prefix += oldPrefix.(string)
	}

	ctx := context.WithValue(parentCtx, prefixKey, prefix)

	return e.WithContext(ctx)
}
