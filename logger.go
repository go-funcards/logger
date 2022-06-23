package logger

import (
	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"strings"
)

// FileLoggerConfig structure represents configuration for the file logger
type FileLoggerConfig struct {
	// Filename is the file to write logs to.  Backup log files will be retained
	// in the same directory.  It uses <processname>-lumberjack.log in
	// os.TempDir() if empty.
	LogOutput string `yaml:"log_output"`

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSize int `yaml:"max_size"`

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAge int `yaml:"max_age"`

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int `yaml:"max_backups"`

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool `yaml:"compress"`
}

func (fl *FileLoggerConfig) InitDefaults() *FileLoggerConfig {
	if fl.LogOutput == "" {
		fl.LogOutput = os.TempDir()
	}

	if fl.MaxSize == 0 {
		fl.MaxSize = 100
	}

	if fl.MaxAge == 0 {
		fl.MaxAge = 24
	}

	if fl.MaxBackups == 0 {
		fl.MaxBackups = 10
	}

	return fl
}

type Config struct {
	// Level is the minimum enabled logging level
	Level string `yaml:"level" env:"LEVEL" env-default:"INFO"`

	// Logger line ending. Default: "\n" for the all modes except production
	LineEnding string `yaml:"line_ending"`

	// Encoding sets the logger's encoding. InitDefault values are "json" and
	// "console", as well as any third-party encodings registered via
	// RegisterEncoder.
	Encoding string `yaml:"encoding"`

	// Output is a list of URLs or file paths to write logging output to.
	// See Open for details.
	Output []string `yaml:"output"`

	// ErrorOutput is a list of URLs to write internal logger errors to.
	// The default is standard error.
	//
	// Note that this setting only affects internal errors; for sample code that
	// sends error-level logs to a different location from info- and debug-level
	// logs, see the package-level AdvancedConfiguration example.
	ErrorOutput []string `yaml:"error_output"`

	// File logger options
	FileLogger *FileLoggerConfig `yaml:"file_logger_options"`
}

// BuildLogger converts config into Zap configuration.
func (cfg *Config) BuildLogger(debug bool) (*zap.Logger, error) {
	if cfg.LineEnding == "" {
		cfg.LineEnding = zapcore.DefaultLineEnding
	}

	var zCfg zap.Config
	if debug {
		zCfg = zap.Config{
			Level:       zap.NewAtomicLevelAt(zap.DebugLevel),
			Development: true,
			Encoding:    "console",
			EncoderConfig: zapcore.EncoderConfig{
				TimeKey:        "ts",
				LevelKey:       "level",
				NameKey:        "logger",
				CallerKey:      zapcore.OmitKey,
				FunctionKey:    zapcore.OmitKey,
				MessageKey:     "msg",
				StacktraceKey:  zapcore.OmitKey,
				LineEnding:     cfg.LineEnding,
				EncodeLevel:    ColoredLevelEncoder,
				EncodeName:     ColoredNameEncoder,
				EncodeTime:     zapcore.ISO8601TimeEncoder,
				EncodeDuration: zapcore.StringDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			},
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		}
	} else {
		zCfg = zap.Config{
			Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
			Development: false,
			Encoding:    "json",
			EncoderConfig: zapcore.EncoderConfig{
				TimeKey:        "ts",
				LevelKey:       "level",
				NameKey:        "logger",
				CallerKey:      zapcore.OmitKey,
				FunctionKey:    zapcore.OmitKey,
				MessageKey:     "msg",
				StacktraceKey:  zapcore.OmitKey,
				LineEnding:     cfg.LineEnding,
				EncodeLevel:    zapcore.LowercaseLevelEncoder,
				EncodeTime:     zapcore.EpochTimeEncoder,
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			},
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		}
	}

	if debug {
		cfg.Level = "DEBUG"
	} else if cfg.Level == "" {
		cfg.Level = "INFO"
	}

	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(cfg.Level)); err == nil {
		zCfg.Level = level
	}

	if cfg.Encoding != "" {
		zCfg.Encoding = cfg.Encoding
	}

	if len(cfg.Output) != 0 {
		zCfg.OutputPaths = cfg.Output
	}

	if len(cfg.ErrorOutput) != 0 {
		zCfg.ErrorOutputPaths = cfg.ErrorOutput
	}

	// if we also have a file logger specified in the config
	// init it
	// otherwise - return standard config
	if cfg.FileLogger != nil {
		// init absent options
		cfg.FileLogger.InitDefaults()

		w := zapcore.AddSync(
			&lumberjack.Logger{
				Filename:   cfg.FileLogger.LogOutput,
				MaxSize:    cfg.FileLogger.MaxSize,
				MaxAge:     cfg.FileLogger.MaxAge,
				MaxBackups: cfg.FileLogger.MaxBackups,
				Compress:   cfg.FileLogger.Compress,
			},
		)

		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zCfg.EncoderConfig),
			w,
			zCfg.Level,
		)
		return zap.New(core), nil
	}

	return zCfg.Build()
}

// ColoredLevelEncoder colorizes log levels.
func ColoredLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch level {
	case zapcore.DebugLevel:
		enc.AppendString(color.HiWhiteString(level.CapitalString()))
	case zapcore.InfoLevel:
		enc.AppendString(color.HiCyanString(level.CapitalString()))
	case zapcore.WarnLevel:
		enc.AppendString(color.HiYellowString(level.CapitalString()))
	case zapcore.ErrorLevel, zapcore.DPanicLevel:
		enc.AppendString(color.HiRedString(level.CapitalString()))
	case zapcore.PanicLevel, zapcore.FatalLevel:
		enc.AppendString(color.HiMagentaString(level.CapitalString()))
	}
}

// ColoredNameEncoder colorizes service names.
func ColoredNameEncoder(s string, enc zapcore.PrimitiveArrayEncoder) {
	if len(s) < 12 {
		s += strings.Repeat(" ", 12-len(s))
	}

	enc.AppendString(color.HiGreenString(s))
}
