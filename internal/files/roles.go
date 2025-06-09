package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"replbac/internal/models"
)

// ReadRoleFile reads and parses a single YAML role file
func ReadRoleFile(filePath string) (models.Role, error) {
	var role models.Role

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".yaml" && ext != ".yml" {
		return role, errors.New("not a YAML file")
	}

	// Read file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		return role, fmt.Errorf("failed to read file: %w", err)
	}

	// Check if file is empty
	if len(data) == 0 {
		return role, errors.New("file is empty")
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, &role); err != nil {
		return role, errors.New("failed to parse YAML")
	}

	// Validate the role
	if err := ValidateRole(role); err != nil {
		return role, err
	}

	return role, nil
}

// FindRoleFiles recursively finds all YAML files in a directory
func FindRoleFiles(rootPath string) ([]string, error) {
	var files []string

	// Check if directory exists
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", rootPath)
	}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if it's a YAML file
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}

// LoadRolesFromDirectory loads all valid role files from a directory recursively
// Invalid files are silently skipped to allow for mixed content directories
func LoadRolesFromDirectory(rootPath string) ([]models.Role, error) {
	// Find all YAML files
	files, err := FindRoleFiles(rootPath)
	if err != nil {
		return nil, err
	}

	var roles []models.Role

	// Load each file, silently skipping invalid ones
	for _, filePath := range files {
		role, err := ReadRoleFile(filePath)
		if err != nil {
			// Skip invalid files silently to allow mixed content
			continue
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// ValidateRole validates that a role has required fields and valid structure
func ValidateRole(role models.Role) error {
	// Check required name field
	if role.Name == "" {
		return errors.New("role name is required")
	}

	// Allow empty resources - some roles might be placeholders or have specific use cases
	return nil
}