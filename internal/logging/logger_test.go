package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogSanitization(t *testing.T) {
	tests := []struct {
		name     string
		logMsg   string
		args     []interface{}
		expected []string    // Strings that should be present in output
		blocked  []string    // Strings that should NOT be present in output
	}{
		{
			name:   "API token in message",
			logMsg: "connecting to API with token: %s",
			args:   []interface{}{"abcdef1234567890abcdef1234567890"},
			expected: []string{"token: [REDACTED]"},
			blocked:  []string{"abcdef1234567890abcdef1234567890"},
		},
		{
			name:   "Authorization header",
			logMsg: "HTTP request headers: Authorization: Bearer %s",
			args:   []interface{}{"abc123def456ghi789"},
			expected: []string{"Authorization: Bearer [REDACTED]"},
			blocked:  []string{"abc123def456ghi789"},
		},
		{
			name:   "API key in URL",
			logMsg: "calling endpoint: https://api.example.com/data?api_key=%s&limit=10",
			args:   []interface{}{"secret-api-key-12345"},
			expected: []string{"api_key=[REDACTED]"},
			blocked:  []string{"secret-api-key-12345"},
		},
		{
			name:   "Password in config",
			logMsg: "config loaded: password=%s database=%s",
			args:   []interface{}{"mySecretPassword", "production_db"},
			expected: []string{"password=[REDACTED]", "production_db"},
			blocked:  []string{"mySecretPassword"},
		},
		{
			name:   "Multiple sensitive values",
			logMsg: "credentials: token=%s secret=%s public_key=%s",
			args:   []interface{}{"longTokenValue12345678901234567890", "mySecret123", "pk_12345"},
			expected: []string{"token=[REDACTED]", "secret=[REDACTED]"},
			blocked:  []string{"longTokenValue12345678901234567890", "mySecret123"},
		},
		{
			name:   "Normal log message unchanged",
			logMsg: "processing %d files in directory %s",
			args:   []interface{}{5, "/home/user/data"},
			expected: []string{"processing 5 files", "/home/user/data"},
			blocked:  []string{},
		},
		{
			name:   "Short strings not redacted",
			logMsg: "user %s logged in with role %s",
			args:   []interface{}{"admin", "viewer"},
			expected: []string{"admin", "viewer"},
			blocked:  []string{"[REDACTED]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(&buf, true)

			logger.Info(tt.logMsg, tt.args...)

			output := buf.String()

			// Check that expected strings are present
			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but got: %s", expected, output)
				}
			}

			// Check that blocked strings are NOT present
			for _, blocked := range tt.blocked {
				if strings.Contains(output, blocked) {
					t.Errorf("Expected output to NOT contain %q, but got: %s", blocked, output)
				}
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	logger := &Logger{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "API token gets redacted",
			input:    "Using token: abcdef1234567890abcdef1234567890",
			expected: "Using token: [REDACTED]",
		},
		{
			name:     "Authorization header gets redacted",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "Authorization: Bearer [REDACTED]",
		},
		{
			name:     "API key in URL gets redacted",
			input:    "https://api.example.com?api_key=secret123&other=value",
			expected: "https://api.example.com?api_key=[REDACTED]&other=value",
		},
		{
			name:     "Password field gets redacted",
			input:    "password=secretpass123 username=admin",
			expected: "password=[REDACTED] username=admin",
		},
		{
			name:     "Normal text unchanged",
			input:    "Processing 10 files from directory /tmp",
			expected: "Processing 10 files from directory /tmp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger.sanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, false) // Non-verbose mode

	// In non-verbose mode (ERROR level), only Error messages should appear
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()

	if strings.Contains(output, "debug message") {
		t.Error("Debug message should not appear in non-verbose mode")
	}
	if strings.Contains(output, "info message") {
		t.Error("Info message should not appear in non-verbose mode")
	}
	if strings.Contains(output, "warn message") {
		t.Error("Warn message should not appear in non-verbose mode")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message should appear in non-verbose mode")
	}
}

func TestVerboseMode(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, true) // Verbose mode

	// In verbose mode (INFO level), Info, Warn and Error messages should appear
	logger.Debug("debug message")
	logger.Info("info message") 
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()

	if strings.Contains(output, "debug message") {
		t.Error("Debug message should not appear even in verbose mode")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Info message should appear in verbose mode")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should appear in verbose mode")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message should appear in verbose mode")
	}
}