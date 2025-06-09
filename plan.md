# replbac TDD Implementation Plan

## Project Overview
Building a Go CLI tool for synchronizing RBAC configurations between local YAML files and Replicated Vendor Portal API using Test-Driven Development.

## Architecture Foundation
- Language: Go
- CLI Framework: Cobra
- Project Structure: golang-standards/project-layout
- Build System: Make
- Testing: Go's built-in testing framework
- HTTP Client: Standard library + custom wrapper

## Development Strategy
1. Start with core data structures and validation
2. Build CLI framework incrementally
3. Add HTTP client and API integration
4. Implement file operations and YAML processing
5. Wire everything together with main command logic
6. Add advanced features (dry-run, init/bootstrap)

---

## Step-by-Step Implementation Prompts

### Step 1: Project Setup and Core Data Structures

```
Set up a new Go project following golang-standards/project-layout structure. Create the basic project files:

1. Initialize go.mod for module `replbac`
2. Create the standard directory structure:
   - cmd/replbac/ (main application)
   - internal/ (private application code)
   - pkg/ (public library code that can be imported)
   - Makefile (build system)
   - .gitignore (Go-specific)

3. Define core data structures in internal/models/:
   - Role struct representing YAML role format
   - APIRole struct representing API JSON format with v1 wrapper
   - Config struct for application configuration
   - Add JSON and YAML struct tags

4. Write unit tests first for:
   - Role validation (name required, resources structure)
   - Conversion between Role and APIRole formats
   - Config validation

5. Implement the structs to make tests pass

Focus on TDD: Write failing tests first, then implement just enough code to make them pass. Keep it simple and focused on data structures only.
```

### Step 2: Configuration Management

```
Implement configuration management system with multiple source support (env vars, config files, CLI flags).

Using TDD approach:

1. Write tests for configuration loading in internal/config/:
   - Test loading from environment variables
   - Test loading from YAML config file
   - Test precedence (CLI flags > env vars > config file > defaults)
   - Test validation of required fields (API endpoints, token sources)

2. Implement ConfigLoader with methods:
   - LoadFromEnv() 
   - LoadFromFile(path string)
   - Merge() for combining sources with proper precedence
   - Validate() for ensuring required fields

3. Add support for these config options:
   - API base URL
   - API token (from multiple sources)
   - Default confirmation behavior
   - Log level

4. Write integration tests that verify the full configuration loading flow

Keep the implementation minimal - just enough to make tests pass. Don't add features not covered by tests.
```

### Step 3: YAML File Operations

```
Implement YAML file reading and processing with comprehensive error handling.

TDD approach:

1. Write tests in internal/files/ for:
   - Reading single YAML role file and parsing to Role struct
   - Recursive directory traversal to find all .yaml/.yml files
   - Error handling for invalid YAML, missing files, permission issues
   - File validation (ensuring required fields like 'name' exist)

2. Implement FileProcessor with methods:
   - ReadRoleFile(path string) (*Role, error)
   - ScanDirectory(path string) ([]string, error) // returns file paths
   - LoadRolesFromDirectory(path string) ([]*Role, error)

3. Add validation logic:
   - Ensure role name is not empty
   - Validate resources structure (allowed/denied arrays)
   - Return meaningful error messages for common issues

4. Test edge cases:
   - Empty directories
   - Mixed file types (should ignore non-YAML)
   - Deeply nested directory structures
   - Files with same role names (should error)

Focus on robust file handling and clear error messages. Implement only what's needed for the tests.
```

### Step 4: HTTP Client and API Integration

```
Build HTTP client wrapper for Replicated Vendor Portal API integration.

TDD with mock server testing:

1. Write tests in internal/api/ for:
   - HTTP client initialization with proper headers (auth token)
   - GET request to list existing roles
   - POST/PUT requests to create/update roles  
   - Error handling for various HTTP status codes (401, 403, 404, 500)
   - Request/response body serialization (Role <-> APIRole conversion)

2. Implement APIClient with methods:
   - NewClient(baseURL, token string) *APIClient
   - ListRoles() ([]*APIRole, error)
   - CreateRole(role *APIRole) error
   - UpdateRole(id string, role *APIRole) error
   - GetRole(id string) (*APIRole, error)

3. Add proper error types:
   - AuthenticationError
   - ValidationError  
   - NetworkError
   - APIError with status codes

4. Use httptest.Server for testing HTTP interactions
   - Mock successful responses
   - Mock various error conditions
   - Test request formatting and authentication headers

Keep HTTP logic simple and focused. Don't implement retry logic or advanced features yet.
```

### Step 5: Basic CLI Framework with Cobra

```
Set up Cobra CLI framework with basic command structure and flag handling.

TDD approach for CLI:

1. Write tests in cmd/replbac/ for:
   - Root command initialization
   - Flag parsing (--dry-run, --confirm/--no-confirm)
   - Help text generation
   - Exit codes for various scenarios

2. Implement basic CLI structure:
   - Root command with global flags
   - Version command
   - Basic flag definitions without logic yet

3. Write tests for command validation:
   - Required arguments (directory path)
   - Flag combinations
   - Help text accuracy

4. Add cobra.Command setup:
   - Root command with Run function (empty for now)
   - Global persistent flags
   - Usage and help text

5. Test CLI parsing with various inputs:
   - Valid flag combinations
   - Invalid flag combinations  
   - Help flag behavior

Focus only on CLI structure and flag parsing. Don't implement actual business logic yet - just ensure the CLI framework is solid.
```

### Step 6: Core Sync Logic (Main Business Logic)

```
Implement the core synchronization logic that ties together file operations and API calls.

TDD with dependency injection:

1. Write tests in internal/sync/ for:
   - Compare local roles vs remote roles (detect adds/updates/no-changes)
   - Sync decision logic (what actions to take)
   - Dry-run mode (show changes without applying)
   - User confirmation prompts
   - Error aggregation and reporting

2. Implement SyncManager with injected dependencies:
   - NewSyncManager(apiClient APIClient, fileProcessor FileProcessor)
   - CompareRoles(local, remote []*Role) *SyncPlan
   - ExecuteSync(plan *SyncPlan, dryRun bool) error
   - PromptForConfirmation(changes []Change) bool

3. Define SyncPlan struct:
   - RolesToCreate []*Role
   - RolesToUpdate []*Role  
   - RolesUnchanged []*Role
   - Summary statistics

4. Test various sync scenarios:
   - All roles are new (create all)
   - Mix of new/updated/unchanged roles
   - Dry-run mode (no API calls made)
   - User confirms/rejects changes
   - API errors during sync

Use interfaces for dependencies to enable easy mocking. Keep sync logic separate from CLI and API concerns.
```

### Step 7: Wire Main Command Implementation

```
Connect all components in the main command implementation with proper error handling and user feedback.

TDD integration testing:

1. Write integration tests in cmd/replbac/ for:
   - Full sync command flow (config -> files -> API -> sync)
   - Error handling at each step with appropriate user messages
   - Dry-run mode end-to-end
   - Configuration loading and validation
   - Logging output and user feedback

2. Implement the main Run function:
   - Load configuration (env vars, config file, flags)
   - Initialize API client with loaded config
   - Initialize file processor and scan directory
   - Create sync manager and execute sync
   - Handle all error cases with user-friendly messages

3. Add proper logging:
   - Progress indicators during sync
   - Summary of changes made
   - Clear error messages
   - Debug logging for troubleshooting

4. Test complete workflows:
   - Successful sync with various scenarios
   - Authentication failures
   - Network errors
   - File reading errors
   - User cancellation

5. Add graceful error handling:
   - Aggregate and report multiple file errors
   - Continue processing after non-fatal errors
   - Clean shutdown on user interruption

Focus on user experience - clear messages, appropriate exit codes, helpful error reporting.
```

### Step 8: Init/Bootstrap Command

```
Implement the init/bootstrap subcommand to pull existing roles from the API and create local YAML files.

TDD approach:

1. Write tests in internal/bootstrap/ for:
   - Fetch all roles from API
   - Convert APIRole format to local YAML format
   - Generate proper file names from role names
   - Create directory structure as needed
   - Handle file conflicts (existing files with same names)
   - Validate generated YAML can be read back correctly

2. Implement BootstrapManager:
   - NewBootstrapManager(apiClient APIClient) *BootstrapManager
   - FetchRemoteRoles() ([]*Role, error)
   - GenerateLocalFiles(roles []*Role, targetDir string) error
   - GenerateFileName(roleName string) string
   - WriteRoleFile(role *Role, path string) error

3. Add init subcommand to CLI:
   - cobra.Command for "init" subcommand
   - Target directory flag/argument
   - Overwrite confirmation logic
   - Progress reporting

4. Test scenarios:
   - Empty target directory (create all files)
   - Existing files (prompt for overwrite)
   - Invalid role names (sanitize for filenames)
   - Network/API errors during fetch
   - File system permission errors

5. Integration test the full init workflow:
   - API fetch -> YAML generation -> file writing -> verification

Ensure generated files can be processed by the main sync command (round-trip compatibility).
```

### Step 9: Advanced Features and Polish

```
Add remaining advanced features and polish the user experience.

TDD for advanced features:

1. Implement enhanced dry-run reporting:
   - Detailed diff output showing exactly what would change
   - Color-coded output (additions/deletions/modifications)
   - Summary statistics
   - Test with various change scenarios

2. Add comprehensive logging:
   - Structured logging with levels (debug, info, warn, error)
   - Log file output option
   - Progress bars for long operations
   - Test log output formatting and levels

3. Improve error handling and recovery:
   - Partial sync success (some roles succeed, others fail)
   - Retry logic for transient API errors
   - Better error categorization and user guidance
   - Test error scenarios and recovery paths

4. Add validation enhancements:
   - Role name uniqueness checking
   - Resource pattern validation
   - API response validation
   - Test various validation scenarios

5. Performance and robustness:
   - Concurrent API requests (with rate limiting)
   - Large directory handling
   - Memory usage optimization
   - Test with realistic data volumes

6. Documentation and help:
   - Comprehensive help text
   - Usage examples
   - Error message improvements
   - Test help text accuracy and completeness

Focus on production readiness - proper error handling, performance, and user experience.
```

### Step 10: Build System and Final Integration

```
Complete the build system, add comprehensive integration tests, and ensure production readiness.

Final integration and build:

1. Complete Makefile with targets:
   - build: compile binary
   - test: run all tests
   - lint: code quality checks
   - clean: cleanup build artifacts
   - install: install binary locally
   - Test all make targets work correctly

2. Add comprehensive integration tests:
   - End-to-end workflow tests with real file system
   - API integration tests (if test API available)
   - CLI integration tests with various flag combinations
   - Error scenario integration tests

3. Add build and release automation:
   - Cross-platform builds (linux, macOS, windows)
   - Version embedding in binary
   - Release artifact generation
   - Test builds work on different platforms

4. Final testing and validation:
   - Test with realistic YAML files and directory structures
   - Validate against actual API (if available in test environment)
   - Performance testing with large numbers of roles
   - Memory leak testing for long-running operations

5. Documentation completion:
   - README with installation and usage instructions
   - Configuration examples
   - Troubleshooting guide
   - Test documentation accuracy

6. Security review:
   - Ensure no secrets in logs
   - Validate token handling security
   - Check for potential security issues in file operations
   - Test error messages don't leak sensitive information

Ensure the final product is production-ready with proper build system, comprehensive testing, and good documentation.
```

---

## Implementation Notes

### Testing Strategy
- Unit tests for all individual components
- Integration tests for component interactions
- End-to-end tests for complete workflows
- Mock external dependencies (HTTP API, file system where appropriate)
- Test error conditions as thoroughly as success cases

### Code Quality
- Follow Go best practices and conventions
- Use interfaces for dependency injection and testability
- Keep functions small and focused
- Comprehensive error handling with meaningful messages
- Proper logging throughout

### Incremental Development
- Each step builds on previous steps
- No orphaned code - everything integrates
- Each step is fully tested before moving to next
- Regular integration testing to catch issues early
- Refactor as needed to maintain clean architecture

This plan ensures a robust, well-tested CLI tool built incrementally with strong foundations.