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
	"flag"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	ctrlruntimelzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func init() {
	Logger = NewDefault().Sugar()
}

var Logger *zap.SugaredLogger

// Options exports a options struct to be used by cmd's
type Options struct {
	// Enable debug logs
	Debug bool
	// Log format (JSON or plain text)
	Format Format
}

func NewDefaultOptions() Options {
	return Options{
		Debug:  false,
		Format: FormatJSON,
	}
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&o.Debug, "log-debug", o.Debug, "Enables debug logging")
	fs.Var(&o.Format, "log-format", "Log format, one of "+AvailableFormats.String())
}

func (o *Options) Validate() error {
	if !AvailableFormats.Contains(o.Format) {
		return fmt.Errorf("invalid log-format specified %q; available: %s", o.Format, AvailableFormats.String())
	}
	return nil
}

type Format string

// String implements the cli.Value and flag.Value interfaces
func (f *Format) String() string {
	return string(*f)
}

// Set implements the cli.Value and flag.Value interfaces
func (f *Format) Set(s string) error {
	switch strings.ToLower(s) {
	case "json":
		*f = FormatJSON
		return nil
	case "console":
		*f = FormatConsole
		return nil
	default:
		return fmt.Errorf("invalid format '%s'", s)
	}
}

type Formats []Format

const (
	FormatJSON    Format = "JSON"
	FormatConsole Format = "Console"
)

var (
	AvailableFormats = Formats{FormatJSON, FormatConsole}
)

func (f Formats) String() string {
	const separator = ", "
	var s string
	for _, format := range f {
		s = s + separator + string(format)
	}
	return strings.TrimPrefix(s, separator)
}

func (f Formats) Contains(s Format) bool {
	for _, format := range f {
		if s == format {
			return true
		}
	}
	return false
}

func New(debug bool, format Format) *zap.Logger {
	// this basically mimics New<type>Config, but with a custom sink
	sink := zapcore.AddSync(os.Stderr)

	// Level - We only support setting Info+ or Debug+
	lvl := zap.NewAtomicLevelAt(zap.InfoLevel)
	if debug {
		lvl = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	encCfg := zap.NewProductionEncoderConfig()
	// Having a dateformat makes it more easy to look at logs outside of something like Kibana
	encCfg.TimeKey = "time"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	// production config encodes durations as a float of the seconds value, but we want a more
	// readable, precise representation
	encCfg.EncodeDuration = zapcore.StringDurationEncoder

	var enc zapcore.Encoder
	if format == FormatJSON {
		enc = zapcore.NewJSONEncoder(encCfg)
	} else if format == FormatConsole {
		enc = zapcore.NewConsoleEncoder(encCfg)
	}

	opts := []zap.Option{
		zap.AddCaller(),
		zap.ErrorOutput(sink),
	}

	coreLog := zapcore.NewCore(&ctrlruntimelzap.KubeAwareEncoder{Encoder: enc}, sink, lvl)
	return zap.New(coreLog, opts...)
}

// NewDefault creates new default logger
func NewDefault() *zap.Logger {
	return New(false, FormatJSON)
}
