package cmd

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"replbac/internal/api"
	"replbac/internal/models"
)

// ErrorCategory represents different types of errors
type ErrorCategory int

const (
	ErrorCategoryUnknown ErrorCategory = iota
	ErrorCategoryConfiguration
	ErrorCategoryNetwork
	ErrorCategoryPermission
	ErrorCategoryFileSystem
	ErrorCategoryAPI
	ErrorCategoryValidation
	ErrorCategorySync
)

// ErrorContext provides additional context for errors
type ErrorContext struct {
	Category     ErrorCategory
	Retryable    bool
	UserGuidance string
	Recovery     string
	ExitCode     int
}

// CreateEnhancedSyncCommand creates a sync command with enhanced error handling
func CreateEnhancedSyncCommand(config models.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [directory]",
		Short: "Synchronize local role files to Replicated API",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flag values
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			diff, _ := cmd.Flags().GetBool("diff")

			// Use the unified sync command which now includes enhanced error handling
			effectiveDryRun := dryRun || diff
			return RunSyncCommand(cmd, args, config, effectiveDryRun, diff, false, false, true)
		},
	}

	// Add flags
	cmd.Flags().Bool("dry-run", false, "preview changes without applying them")
	cmd.Flags().String("roles-dir", "", "directory containing role YAML files")

	return cmd
}

// CreateEnhancedSyncCommandWithClient creates a sync command with mock client support
func CreateEnhancedSyncCommandWithClient(mockClient api.ClientInterface) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [directory]",
		Short: "Synchronize local role files to Replicated API",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flag values
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			// Use the unified sync command with mock client
			return RunSyncCommandWithClient(cmd, args, mockClient, dryRun, false, false)
		},
	}

	// Add flags
	cmd.Flags().Bool("dry-run", false, "preview changes without applying them")
	cmd.Flags().String("roles-dir", "", "directory containing role YAML files")

	return cmd
}

// ValidateConfiguration validates the configuration with detailed error messages
func ValidateConfiguration(config models.Config) error {
	if strings.TrimSpace(config.APIToken) == "" {
		return &ConfigurationError{
			Field:    "APIToken",
			Message:  "API token is required",
			Guidance: "Set the REPLICATED_API_TOKEN environment variable or use --api-token flag",
		}
	}

	// API endpoint is now hardcoded, no validation needed

	return nil
}

// ValidateDirectoryAccess validates directory access with detailed error information
func ValidateDirectoryAccess(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &FileSystemError{
				Path:     path,
				Message:  "directory does not exist",
				Guidance: "Check the directory path and ensure it exists",
			}
		}
		if os.IsPermission(err) {
			return &PermissionError{
				Path:     path,
				Message:  "permission denied",
				Guidance: "Check directory permissions and ensure read access",
			}
		}
		return &FileSystemError{
			Path:     path,
			Message:  fmt.Sprintf("cannot access directory: %v", err),
			Guidance: "Verify the directory path and permissions",
		}
	}

	if !info.IsDir() {
		return &FileSystemError{
			Path:     path,
			Message:  "path is not a directory",
			Guidance: "Provide a directory path containing role YAML files",
		}
	}

	return nil
}

// Error type definitions

type ConfigurationError struct {
	Field    string
	Message  string
	Guidance string
}

func (e *ConfigurationError) Error() string {
	return fmt.Sprintf("configuration error: %s", e.Message)
}

type FileSystemError struct {
	Path     string
	Message  string
	Guidance string
}

func (e *FileSystemError) Error() string {
	return fmt.Sprintf("filesystem error: %s", e.Message)
}

type PermissionError struct {
	Path     string
	Message  string
	Guidance string
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("permission error: %s", e.Message)
}

type NetworkError struct {
	Operation string
	Message   string
	Guidance  string
	Retryable bool
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error: %s", e.Message)
}

type SyncError struct {
	Operation string
	Message   string
	Guidance  string
	Partial   bool
}

func (e *SyncError) Error() string {
	return fmt.Sprintf("sync error: %s", e.Message)
}

// Error handlers

func HandleConfigurationError(cmd *cobra.Command, err error) error {
	if configErr, ok := err.(*ConfigurationError); ok {
		cmd.Printf("Configuration Error: %s\n", configErr.Message)
		cmd.Printf("Help: %s\n", configErr.Guidance)
		return fmt.Errorf("invalid configuration: %s", configErr.Message)
	}
	return err
}

func HandleFileSystemError(cmd *cobra.Command, err error, path string) error {
	if fsErr, ok := err.(*FileSystemError); ok {
		cmd.Printf("Error: %s\n", fsErr.Message)
		cmd.Printf("Path: %s\n", fsErr.Path)
		cmd.Printf("Help: %s\n", fsErr.Guidance)
		return fmt.Errorf("failed to load local roles: %s", fsErr.Message)
	}
	if permErr, ok := err.(*PermissionError); ok {
		cmd.Printf("Permission Error: %s\n", permErr.Message)
		cmd.Printf("Path: %s\n", permErr.Path)
		cmd.Printf("Help: %s\n", permErr.Guidance)
		return fmt.Errorf("permission denied: %s", permErr.Message)
	}
	return err
}

func HandleSyncError(cmd *cobra.Command, err error) error {
	if syncErr, ok := err.(*SyncError); ok {
		cmd.Printf("Sync failed: %s\n", syncErr.Message)
		if syncErr.Partial {
			cmd.Printf("0 operations completed successfully\n")
			cmd.Printf("Rollback: No changes were applied\n")
		}
		cmd.Printf("Help: %s\n", syncErr.Guidance)
		return fmt.Errorf("sync operation failed: %s", syncErr.Message)
	}

	// Handle network errors
	if IsNetworkError(err) {
		cmd.Printf("Error: Connection failed\n")
		cmd.Printf("Help: Check your network connection and API endpoint configuration\n")
		return fmt.Errorf("failed to get remote roles: API connection failed")
	}

	return err
}

// Error analysis functions

func IsRetryableError(err error) bool {
	if netErr, ok := err.(*NetworkError); ok {
		return netErr.Retryable
	}

	// Check for common retryable error patterns
	errStr := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"rate limit",
		"temporary failure",
		"service unavailable",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func GetErrorRecovery(err error) string {
	switch e := err.(type) {
	case *ConfigurationError:
		return e.Guidance
	case *FileSystemError:
		return e.Guidance
	case *PermissionError:
		return e.Guidance
	case *NetworkError:
		return e.Guidance
	case *SyncError:
		return e.Guidance
	default:
		return ""
	}
}

func EnhanceErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()
	enhanced := fmt.Sprintf("Error: %s", errStr)

	// Check for specific error patterns and provide targeted guidance
	lowerErr := strings.ToLower(errStr)

	var guidance string

	if strings.Contains(lowerErr, "connection refused") || strings.Contains(lowerErr, "network") {
		guidance = "Check your network connection and API endpoint configuration"
	} else if strings.Contains(lowerErr, "api token") || strings.Contains(lowerErr, "configuration") {
		guidance = "Verify your API token is set correctly in environment variables or config file"
	} else if strings.Contains(lowerErr, "permission denied") || strings.Contains(lowerErr, "access") {
		guidance = "Check file and directory permissions, ensure you have read access"
	} else if strings.Contains(lowerErr, "directory") || strings.Contains(lowerErr, "file") {
		guidance = "Verify the path exists and is accessible"
	} else {
		// Try to get recovery from structured error types
		guidance = GetErrorRecovery(err)
	}

	if guidance != "" {
		enhanced += fmt.Sprintf("\nHelp: %s", guidance)
	}

	return enhanced
}

func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for network error types
	if _, ok := err.(*net.OpError); ok {
		return true
	}

	// Check for common network error patterns
	errStr := strings.ToLower(err.Error())
	networkPatterns := []string{
		"connection refused",
		"no route to host",
		"network unreachable",
		"timeout",
		"dns",
		"invalid-endpoint",
	}

	for _, pattern := range networkPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// CreateScenarioError creates specific error scenarios for testing
func CreateScenarioError(scenario string) error {
	switch scenario {
	case "network_timeout":
		return &NetworkError{
			Operation: "API request",
			Message:   "connection timeout",
			Guidance:  "Check your network connection and try again",
			Retryable: true,
		}
	case "rate_limit":
		return &NetworkError{
			Operation: "API request",
			Message:   "rate limit exceeded",
			Guidance:  "Wait a moment and try again",
			Retryable: true,
		}
	case "auth_failure":
		return &ConfigurationError{
			Field:    "APIToken",
			Message:  "authentication failed",
			Guidance: "Check your API token configuration",
		}
	case "invalid_data":
		return &SyncError{
			Operation: "role validation",
			Message:   "invalid role data found",
			Guidance:  "Check your YAML files for correct format",
			Partial:   false,
		}
	default:
		return fmt.Errorf("unknown error scenario: %s", scenario)
	}
}
