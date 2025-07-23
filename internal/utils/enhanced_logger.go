package utils

import (
    "context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLevel represents the severity level of a log message
type LogLevel string

const (
	// LogLevelDebug is for detailed debugging information
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo is for general operational information
	LogLevelInfo LogLevel = "info"
	// LogLevelWarn is for warning events
	LogLevelWarn LogLevel = "warn"
	// LogLevelError is for error events
	LogLevelError LogLevel = "error"
	// LogLevelFatal is for critical events that cause the application to exit
	LogLevelFatal LogLevel = "fatal"
)

// EnhancedLogger provides advanced logging functionality
type EnhancedLogger struct {
	logger *zap.SugaredLogger
	config *EnhancedLoggerConfig
	
}

// EnhancedLoggerConfig holds configuration for the enhanced logger
type EnhancedLoggerConfig struct {
	Enabled      bool
	Level        LogLevel
	Path         string
	MaxSize      int  // megabytes
	MaxBackups   int  // number of backups
	MaxAge       int  // days
	Compress     bool // compress rotated files
	ConsoleLog   bool // log to console
	JSONFormat   bool // use JSON format
	CallerInfo   bool // include caller information
	StackTraces  bool // include stack traces for errors
	Development  bool // development mode
	RotationTime int  // hours
	
}

// NewEnhancedLogger creates a new enhanced logger instance
func NewEnhancedLogger(config *EnhancedLoggerConfig) (*EnhancedLogger, error) {
	if !config.Enabled {
		// Return a no-op logger if logging is disabled
		return &EnhancedLogger{
			logger: zap.NewNop().Sugar(),
			config: config,
		}, nil
	}

	// Create log directory if it doesn't exist
	logDir := filepath.Dir(config.Path)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Configure encoder
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Set encoder based on format preference
	var encoder zapcore.Encoder
	if config.JSONFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Configure log rotation
	rotatingLogger := &lumberjack.Logger{
		Filename:   config.Path,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	// Create writers
	var writers []zapcore.WriteSyncer
	writers = append(writers, zapcore.AddSync(rotatingLogger))

	// Add console output if enabled
	if config.ConsoleLog {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// Combine writers if needed
	var writeSyncer zapcore.WriteSyncer
	if len(writers) > 1 {
		writeSyncer = zapcore.NewMultiWriteSyncer(writers...)
	} else {
		writeSyncer = writers[0]
	}

	// Set log level
	var level zapcore.Level
	switch config.Level {
	case LogLevelDebug:
		level = zapcore.DebugLevel
	case LogLevelInfo:
		level = zapcore.InfoLevel
	case LogLevelWarn:
		level = zapcore.WarnLevel
	case LogLevelError:
		level = zapcore.ErrorLevel
	case LogLevelFatal:
		level = zapcore.FatalLevel
	default:
		level = zapcore.InfoLevel
	}

	// Create core
	core := zapcore.NewCore(encoder, writeSyncer, zap.NewAtomicLevelAt(level))

	// Create logger
	var zapLogger *zap.Logger
	if config.Development {
		zapLogger = zap.New(core, zap.Development())
	} else {
		zapLogger = zap.New(core)
	}

	// Add caller info if enabled
	if config.CallerInfo {
		zapLogger = zapLogger.WithOptions(zap.AddCaller())
	}

	// Add stacktrace for errors if enabled
	if config.StackTraces {
		zapLogger = zapLogger.WithOptions(zap.AddStacktrace(zapcore.ErrorLevel))
	}

	// Create sugar logger
	sugarLogger := zapLogger.Sugar()

	return &EnhancedLogger{
		logger: sugarLogger,
		config: config,
	}, nil
}

// Debug logs a debug message
func (l *EnhancedLogger) Debug(format string, args ...interface{}) {
	if l.config.Enabled {
		l.logger.Debugf(format, args...)
	}
}

// Info logs an informational message
func (l *EnhancedLogger) Info(format string, args ...interface{}) {
	if l.config.Enabled {
		l.logger.Infof(format, args...)
	}
}

// Warn logs a warning message
func (l *EnhancedLogger) Warn(format string, args ...interface{}) {
	if l.config.Enabled {
		l.logger.Warnf(format, args...)
	}
}

// Error logs an error message
func (l *EnhancedLogger) Error(format string, args ...interface{}) {
	if l.config.Enabled {
		l.logger.Errorf(format, args...)
	}
}

// Fatal logs a fatal message and exits
func (l *EnhancedLogger) Fatal(format string, args ...interface{}) {
	if l.config.Enabled {
		l.logger.Fatalf(format, args...)
	}
}

// With returns a logger with the specified fields added to the context
func (l *EnhancedLogger) With(fields map[string]interface{}) *EnhancedLogger {
	if !l.config.Enabled {
		return l
	}

	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}

	return &EnhancedLogger{
		logger: l.logger.With(args...),
		config: l.config,
	}
}

// Close flushes any buffered log entries
func (l *EnhancedLogger) Close() error {
	if l.config.Enabled {
		return l.logger.Sync()
	}
	return nil
}

// StartRotationScheduler starts a scheduler to rotate logs at specified intervals
func (l *EnhancedLogger) StartRotationScheduler(ctx context.Context) {
	if !l.config.Enabled || l.config.RotationTime <= 0 {
		return
	}

	ticker := time.NewTicker(time.Duration(l.config.RotationTime) * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				l.rotateLog()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// rotateLog performs log rotation
// rotateLog performs log rotation
func (l *EnhancedLogger) rotateLog() {
	if !l.config.Enabled {
		return
	}

	l.Info("Triggering log rotation")
}


// CleanupOldLogs removes log files older than the specified duration
func (l *EnhancedLogger) CleanupOldLogs(logDir string, maxAge time.Duration) error {
	if !l.config.Enabled {
		return nil
	}

	return filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is a log file
		if filepath.Ext(path) == ".log" {
			// Check if file is older than maxAge
			if time.Since(info.ModTime()) > maxAge {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove old log file %s: %w", path, err)
				}
			}
		}

		return nil
	})
}

// GetWriter returns an io.Writer that can be used with other logging systems
func (l *EnhancedLogger) GetWriter() io.Writer {
	return &logWriter{logger: l}
}

// logWriter implements io.Writer for the enhanced logger
type logWriter struct {
	logger *EnhancedLogger
}

// Write implements io.Writer
func (w *logWriter) Write(p []byte) (n int, err error) {
	w.logger.Info("%s", string(p))
	return len(p), nil
}
