package logging

import (
	"fmt"
	"io"
	"time"
)

// LogLevel represents the level of logging
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// Logger provides structured logging for the application
type Logger struct {
	output  io.Writer
	level   LogLevel
	verbose bool
}

// NewLogger creates a new logger instance that outputs to stderr by default
// Default log level is ERROR for quiet operation
func NewLogger(output io.Writer, verbose bool) *Logger {
	level := ErrorLevel // Default to ERROR level for quiet operation
	if verbose {
		level = InfoLevel // --verbose enables INFO level
	}
	
	return &Logger{
		output:  output,
		level:   level,
		verbose: verbose,
	}
}

// NewDebugLogger creates a logger with DEBUG level enabled
func NewDebugLogger(output io.Writer) *Logger {
	return &Logger{
		output:  output,
		level:   DebugLevel,
		verbose: true,
	}
}

// Debug logs debug-level messages (only shown in verbose mode)
func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.level <= DebugLevel {
		l.log("DEBUG", msg, args...)
	}
}

// Info logs informational messages
func (l *Logger) Info(msg string, args ...interface{}) {
	if l.level <= InfoLevel {
		l.log("INFO", msg, args...)
	}
}

// Warn logs warning messages
func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.level <= WarnLevel {
		l.log("WARN", msg, args...)
	}
}

// Error logs error messages
func (l *Logger) Error(msg string, args ...interface{}) {
	if l.level <= ErrorLevel {
		l.log("ERROR", msg, args...)
	}
}


// TimedOperation tracks and logs the duration of an operation
func (l *Logger) TimedOperation(operation string, fn func() error) error {
	l.Info("starting %s", operation)
	start := time.Now()
	
	err := fn()
	duration := time.Since(start)
	
	if err != nil {
		l.Error("%s failed after %v: %v", operation, duration, err)
	} else {
		l.Info("%s completed in %v", operation, duration)
	}
	
	return err
}

// log formats and writes log messages
func (l *Logger) log(level, msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	formattedMsg := fmt.Sprintf(msg, args...)
	fmt.Fprintf(l.output, "[%s] %s %s\n", level, timestamp, formattedMsg)
}

// IsVerbose returns whether verbose logging is enabled
func (l *Logger) IsVerbose() bool {
	return l.verbose
}