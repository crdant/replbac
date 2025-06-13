package logging

import (
	"fmt"
	"io"
	"regexp"
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

// log formats and writes log messages with sensitive data sanitization
func (l *Logger) log(level, msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")

	// First format the message with original args
	formattedMsg := fmt.Sprintf(msg, args...)

	// Then sanitize the complete formatted message
	sanitizedMsg := l.sanitizeString(formattedMsg)

	fmt.Fprintf(l.output, "[%s] %s %s\n", level, timestamp, sanitizedMsg)
}

// sanitizeString removes or masks sensitive data patterns in strings
func (l *Logger) sanitizeString(s string) string {
	// Patterns for sensitive data - most specific first to avoid conflicts
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// Complete Authorization header with Bearer token (full match)
		{regexp.MustCompile(`(?i)authorization:\s*bearer\s+[A-Za-z0-9\-_.=]+`), "Authorization: Bearer [REDACTED]"},

		// URL query parameters with sensitive names
		{regexp.MustCompile(`(?i)([?&](api[_-]?key|token|secret|password|pass)=)[^&\s\n]+`), "${1}[REDACTED]"},

		// Key-value patterns with equals (not in URLs)
		{regexp.MustCompile(`(?i)(^|[^?&])(api[_-]?key|apikey|token|secret|password|pass)=\s*[^\s\n&,}]+`), "${1}${2}=[REDACTED]"},

		// Key-value patterns with colon (config style)
		{regexp.MustCompile(`(?i)(api[_-]?key|apikey|token|secret|password|pass):\s*[^\s\n,}]+`), "${1}: [REDACTED]"},

		// Long alphanumeric strings (likely tokens) - but only after colon or space
		{regexp.MustCompile(`(\s|:\s*)[A-Za-z0-9_\-]{20,}(\s|$)`), "${1}[REDACTED_TOKEN]${2}"},
	}

	result := s
	for _, pattern := range patterns {
		result = pattern.regex.ReplaceAllString(result, pattern.replacement)
	}

	return result
}

// IsVerbose returns whether verbose logging is enabled
func (l *Logger) IsVerbose() bool {
	return l.verbose
}
