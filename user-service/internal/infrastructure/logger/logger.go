package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Nezent/microservice-template/user-service/config"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger interface {
	fxevent.Logger
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	Panic(msg string, fields ...zap.Field)
	With(fields ...zap.Field) Logger
	Named(name string) Logger
	Sync() error
}

type LogFormat string

const (
	JSONFormat LogFormat = "json"
	TextFormat LogFormat = "text"
)

type zapLogger struct {
	*zap.Logger
}

var _ Logger = (*zapLogger)(nil)

// NewLogger creates a new zap-based logger from the provided configuration.
func NewLogger(config config.LogConfig) (Logger, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid log config: %w", err)
	}

	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", config.Level, err)
	}

	encCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if config.Format == string(TextFormat) {
		encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encCfg.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
	}
	if config.DisableTimestamp {
		encCfg.TimeKey = ""
	}

	// Optional time encoding override for performance-sensitive deployments
	switch config.TimeEncoding {
	case "iso8601":
		encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	case "epoch":
		encCfg.EncodeTime = zapcore.EpochTimeEncoder
	case "epoch_millis":
		encCfg.EncodeTime = zapcore.EpochMillisTimeEncoder
	case "epoch_nanos":
		encCfg.EncodeTime = zapcore.EpochNanosTimeEncoder
	case "":
		// keep defaults above
	default:
		// validated earlier; ignore unknown to be safe
	}

	var encoder zapcore.Encoder
	switch config.Format {
	case string(JSONFormat):
		encoder = zapcore.NewJSONEncoder(encCfg)
	case string(TextFormat):
		encoder = zapcore.NewConsoleEncoder(encCfg)
	default:
		return nil, fmt.Errorf("unsupported log format: %s", config.Format)
	}

	writers, err := createWriters(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create writers: %w", err)
	}

	// Lock writers to avoid interleaved output in concurrent scenarios
	for i := range writers {
		writers[i] = zapcore.Lock(writers[i])
	}
	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writers...), level)

	// Enable sampling to reduce overhead under high-frequency logging, if configured
	if config.Sampling.Enabled {
		tick := config.Sampling.Tick
		if tick == 0 {
			tick = time.Second
		}
		initial := getOrDefault(config.Sampling.Initial, 100)
		thereafter := getOrDefault(config.Sampling.Thereafter, 100)
		core = zapcore.NewSamplerWithOptions(core, tick, initial, thereafter)
	}

	opts := []zap.Option{
		zap.ErrorOutput(zapcore.Lock(os.Stderr)),
	}
	if !config.DisableCaller {
		opts = append(opts, zap.AddCaller())
	}
	if !config.DisableStacktrace {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	logger := zap.New(core, opts...)
	return &zapLogger{Logger: logger}, nil
}

func createWriters(config config.LogConfig) ([]zapcore.WriteSyncer, error) {
	var writers []zapcore.WriteSyncer
	writers = append(writers, zapcore.AddSync(os.Stdout))

	if config.File.Path != "" {
		if err := os.MkdirAll(filepath.Dir(config.File.Path), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		fileWriter := &lumberjack.Logger{
			Filename:   config.File.Path,
			MaxSize:    getOrDefault(config.File.MaxSize, 100),
			MaxAge:     getOrDefault(config.File.MaxDays, 30),
			MaxBackups: getOrDefault(config.File.MaxBackups, 10),
			LocalTime:  config.File.LocalTime,
			Compress:   config.File.Compress,
		}
		writers = append(writers, zapcore.AddSync(fileWriter))
	}

	return writers, nil
}

func getOrDefault(val, def int) int {
	if val == 0 {
		return def
	}
	return val
}

// Logger methods
func (l *zapLogger) Info(msg string, fields ...zap.Field)  { l.Logger.Info(msg, fields...) }
func (l *zapLogger) Error(msg string, fields ...zap.Field) { l.Logger.Error(msg, fields...) }
func (l *zapLogger) Warn(msg string, fields ...zap.Field)  { l.Logger.Warn(msg, fields...) }
func (l *zapLogger) Debug(msg string, fields ...zap.Field) { l.Logger.Debug(msg, fields...) }
func (l *zapLogger) Fatal(msg string, fields ...zap.Field) { l.Logger.Fatal(msg, fields...) }
func (l *zapLogger) Panic(msg string, fields ...zap.Field) { l.Logger.Panic(msg, fields...) }

func (l *zapLogger) With(fields ...zap.Field) Logger {
	return &zapLogger{Logger: l.Logger.With(fields...)}
}

func (l *zapLogger) Named(name string) Logger {
	return &zapLogger{Logger: l.Logger.Named(name)}
}

func (l *zapLogger) Sync() error {
	return l.Logger.Sync()
}

// LogEvent implements fxevent.Logger interface for Fx integration
func (l *zapLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.Logger.Info("OnStart hook executing",
			zap.String("function", e.FunctionName),
			zap.String("caller", e.CallerName),
		)
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.Logger.Error("OnStart hook failed",
				zap.String("function", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.Error(e.Err),
				zap.Duration("runtime", e.Runtime),
			)
		} else {
			l.Logger.Info("OnStart hook executed",
				zap.String("function", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.Duration("runtime", e.Runtime),
			)
		}
	case *fxevent.OnStopExecuting:
		l.Logger.Info("OnStop hook executing",
			zap.String("function", e.FunctionName),
			zap.String("caller", e.CallerName),
		)
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.Logger.Error("OnStop hook failed",
				zap.String("function", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.Error(e.Err),
				zap.Duration("runtime", e.Runtime),
			)
		} else {
			l.Logger.Info("OnStop hook executed",
				zap.String("function", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.Duration("runtime", e.Runtime),
			)
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			l.Logger.Error("Supplied failed",
				zap.String("type", e.TypeName),
				zap.Strings("moduletrace", e.ModuleTrace),
				zap.String("module", e.ModuleName),
				zap.Error(e.Err),
			)
		} else if l.Logger.Core().Enabled(zapcore.DebugLevel) {
			l.Logger.Debug("Supplied",
				zap.String("type", e.TypeName),
				zap.Strings("moduletrace", e.ModuleTrace),
				zap.Strings("stacktrace", e.StackTrace),
				zap.String("module", e.ModuleName),
			)
		}
	case *fxevent.Provided:
		if e.Err != nil {
			l.Logger.Error("Provided failed",
				zap.String("module", e.ModuleName),
				zap.Strings("moduletrace", e.ModuleTrace),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Error(e.Err),
				zap.Strings("types", e.OutputTypeNames),
			)
		} else if l.Logger.Core().Enabled(zapcore.DebugLevel) {
			for _, rtype := range e.OutputTypeNames {
				l.Logger.Debug("Provided",
					zap.String("constructor", e.ConstructorName),
					zap.String("module", e.ModuleName),
					zap.Strings("moduletrace", e.ModuleTrace),
					zap.Strings("stacktrace", e.StackTrace),
					zap.String("type", rtype),
					zap.Bool("private", e.Private),
				)
			}
		}
	case *fxevent.Replaced:
		if e.Err != nil {
			l.Logger.Error("Replaced failed",
				zap.String("module", e.ModuleName),
				zap.Strings("moduletrace", e.ModuleTrace),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Error(e.Err),
				zap.Strings("types", e.OutputTypeNames),
			)
		} else if l.Logger.Core().Enabled(zapcore.DebugLevel) {
			for _, rtype := range e.OutputTypeNames {
				l.Logger.Debug("Replaced",
					zap.String("module", e.ModuleName),
					zap.Strings("moduletrace", e.ModuleTrace),
					zap.Strings("stacktrace", e.StackTrace),
					zap.String("type", rtype),
				)
			}
		}
	case *fxevent.Decorated:
		if e.Err != nil {
			l.Logger.Error("Decorated failed",
				zap.String("module", e.ModuleName),
				zap.Strings("moduletrace", e.ModuleTrace),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Error(e.Err),
				zap.Strings("types", e.OutputTypeNames),
			)
		} else if l.Logger.Core().Enabled(zapcore.DebugLevel) {
			for _, rtype := range e.OutputTypeNames {
				l.Logger.Debug("Decorated",
					zap.String("decorator", e.DecoratorName),
					zap.String("module", e.ModuleName),
					zap.Strings("moduletrace", e.ModuleTrace),
					zap.Strings("stacktrace", e.StackTrace),
					zap.String("type", rtype),
				)
			}
		}
	case *fxevent.BeforeRun:
		l.Logger.Info("Before run",
			zap.String("name", e.Name),
			zap.String("kind", e.Kind),
			zap.String("module", e.ModuleName),
		)
	case *fxevent.Run:
		if e.Err != nil {
			l.Logger.Error("Run failed",
				zap.String("name", e.Name),
				zap.String("kind", e.Kind),
				zap.String("module", e.ModuleName),
				zap.Error(e.Err),
			)
		} else {
			l.Logger.Info("Run succeeded",
				zap.String("name", e.Name),
				zap.String("kind", e.Kind),
				zap.String("module", e.ModuleName),
				zap.Duration("runtime", e.Runtime),
			)
		}
	case *fxevent.Invoking:
		if l.Logger.Core().Enabled(zapcore.DebugLevel) {
			l.Logger.Debug("Invoking",
				zap.String("function", e.FunctionName),
				zap.String("module", e.ModuleName),
			)
		}
	case *fxevent.Invoked:
		if e.Err != nil {
			l.Logger.Error("Invoke failed",
				zap.String("function", e.FunctionName),
				zap.String("module", e.ModuleName),
				zap.Error(e.Err),
			)
		} else if l.Logger.Core().Enabled(zapcore.DebugLevel) {
			l.Logger.Debug("Invoked",
				zap.String("function", e.FunctionName),
				zap.String("module", e.ModuleName),
			)
		}
	case *fxevent.Stopping:
		l.Logger.Info("Received signal",
			zap.String("signal", strings.ToUpper(e.Signal.String())),
		)
	case *fxevent.Stopped:
		if e.Err != nil {
			l.Logger.Error("Stop failed", zap.Error(e.Err))
		} else {
			l.Logger.Info("Stopped")
		}
	case *fxevent.RollingBack:
		l.Logger.Error("Start failed, rolling back", zap.Error(e.StartErr))
	case *fxevent.RolledBack:
		if e.Err != nil {
			l.Logger.Error("Rollback failed", zap.Error(e.Err))
		} else {
			l.Logger.Info("Rolled back")
		}
	case *fxevent.Started:
		if e.Err != nil {
			l.Logger.Error("Start failed", zap.Error(e.Err))
		} else {
			l.Logger.Info("Started")
		}
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			l.Logger.Error("Custom logger initialization failed", zap.Error(e.Err))
		} else if l.Logger.Core().Enabled(zapcore.DebugLevel) {
			l.Logger.Debug("Initialized custom fxevent.Logger", zap.String("function", e.ConstructorName))
		}
	default:
		l.Logger.Warn("Unknown Fx event", zap.String("type", fmt.Sprintf("%T", event)), zap.Reflect("event", event))
	}
}

// ProvideLogger provides a logger instance for dependency injection.
func ProvideLogger(cfg *config.Config) (Logger, error) {
	return NewLogger(config.LogConfig{
		Level:             cfg.Log.Level,
		Format:            cfg.Log.Format,
		DisableTimestamp:  cfg.Log.DisableTimestamp,
		DisableCaller:     cfg.Log.DisableCaller,
		DisableStacktrace: cfg.Log.DisableStacktrace,
		TimeEncoding:      cfg.Log.TimeEncoding,
		Sampling:          cfg.Log.Sampling,
		File: config.LogFileConfig{
			Path:       cfg.Log.File.Path,
			MaxSize:    cfg.Log.File.MaxSize,
			MaxDays:    cfg.Log.File.MaxDays,
			MaxBackups: cfg.Log.File.MaxBackups,
			Compress:   cfg.Log.File.Compress,
			LocalTime:  cfg.Log.File.LocalTime,
		},
	})
}
