package log

import (
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type Format string

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

func (f Formats) Has(s string) bool {
	for _, format := range f {
		if s == string(format) {
			return true
		}
	}
	return false
}

func New(debug bool, format Format) logr.Logger {
	// this basically mimics New<type>Config, but with a custom sink
	sink := zapcore.AddSync(os.Stderr)

	// Level - We only support setting Info+ or Debug+
	// We could potentially use support the full level range, but zap starts with -1(Debug) and ends with 6(fatal)
	// zapr inverts the level: 1 (debug) -6(fatal) to adhere the logr definition (The higher the level the more verbose it is)
	lvl := zap.NewAtomicLevelAt(zap.InfoLevel)
	if debug {
		lvl = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	encCfg := zap.NewProductionEncoderConfig()
	// Having a dateformat makes it more easy to look at logs outside of something like Kibana
	encCfg.TimeKey = "time"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var enc zapcore.Encoder
	if format == FormatJSON {
		enc = zapcore.NewJSONEncoder(encCfg)
	} else if format == FormatConsole {
		enc = zapcore.NewConsoleEncoder(encCfg)
	}

	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.ErrorOutput(sink),
	}

	coreLog := zapcore.NewCore(&ctrlruntimelog.KubeAwareEncoder{Encoder: enc}, sink, lvl)
	log := zap.New(coreLog, opts...)
	return zapr.NewLogger(log)
}
