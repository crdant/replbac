# replbac TODO Tracker

## Current Status: Ready to Begin Implementation

### Completed
- ✅ Project specification analysis
- ✅ Detailed TDD implementation plan created
- ✅ Step-by-step development prompts prepared

### Next Steps

#### Step 1: Project Setup and Core Data Structures
- [x] Initialize Go project with proper module structure
- [x] Create golang-standards/project-layout directory structure
- [x] Define core data structures (Role, APIRole, Config)
- [x] Write and implement unit tests for data structures
- [x] Set up basic Makefile and .gitignore

#### Step 2: Configuration Management
- [x] Implement configuration loading from multiple sources
- [x] Add support for environment variables, config files, CLI flags
- [x] Test configuration precedence and validation
- [x] Handle API token management securely

#### Step 3: YAML File Operations
- [x] Implement YAML file reading and parsing
- [x] Add recursive directory traversal
- [x] Build robust error handling for file operations
- [x] Test edge cases and validation

#### Step 4: HTTP Client and API Integration
- [x] Build HTTP client wrapper for Replicated API
- [x] Implement role CRUD operations
- [x] Add proper error handling and types
- [x] Test with mock HTTP server

#### Step 5: Basic CLI Framework
- [x] Set up Cobra CLI framework
- [x] Implement flag parsing and validation
- [x] Add help text and usage information
- [x] Test CLI structure and commands

#### Step 6: Core Sync Logic
- [x] Implement role comparison logic
- [x] Build sync planning and execution
- [x] Add dry-run mode support
- [x] Test various sync scenarios

#### Step 7: Wire Main Command
- [x] Connect all components in main command
- [x] Add comprehensive error handling
- [x] Implement user feedback and logging
- [x] Test complete workflows

#### Step 8: Init/Bootstrap Command
- [x] Implement API-to-local sync (reverse direction)
- [x] Add file generation from API roles
- [x] Handle conflicts and user prompts
- [x] Test round-trip compatibility (using our mocks)

#### Step 9: Advanced Features
- [x] Enhanced dry-run reporting with diffs
- [x] Comprehensive logging system
- [x] Production readiness improvements

#### Step 10: Ergonomics and User Experience
- [x] Rename `init` command to `pull` and add dry-run, confirm, and diff flags
- [x] Remove `--output-dir` flag and `--roles-dir` flags since we have a positional argument
- [x] Use a flag to for whether to delete roles not in local YAML files (default to false)
- [x] Document environment variables in help alongside the equivalent CLI flags
- [x] Let's get rid of the configuration for the API endpoint since there's only one
- [x] Determine if we should make any of our package public and add to do items to this document for each one under Step 12
- [x] Add man page

#### Step 11: Add membership
- [ ] Add `members` field to Role model in `internal/models/models.go`
- [ ] Update YAML parsing and validation in `internal/roles/files.go` to handle members field
- [ ] Add team member data structures (Member, TeamMember) to models
- [ ] Implement GET team members API call in `internal/api/client.go`
- [ ] Implement PUT team member role assignment API call in `internal/api/client.go`
- [ ] Add member validation logic to ensure users only appear in one role at a time
- [ ] Update sync comparison logic in `internal/sync/compare.go` to handle member assignments
- [ ] Add member assignment execution to `internal/sync/executor.go`
- [ ] Update CLI commands to support member sync operations
- [ ] Add comprehensive tests for member management functionality
- [ ] Update documentation and examples to show members field usage

#### Step 12: Expose Public API
- [ ] Move `internal/models` to `pkg/models` - Core data structures for roles and API communication
- [ ] Move `internal/api` to `pkg/api` with logger interface - HTTP client for Replicated API operations
- [ ] Move `internal/roles` to `pkg/roles` with optional logger interface - YAML file operations for role management
- [ ] Move `internal/sync` to `pkg/sync` with logger interface - Role comparison and sync planning algorithms
- [ ] Keep `internal/config` internal - CLI-specific configuration loading
- [ ] Keep `internal/logging` internal - Use interfaces in public packages instead
- [ ] Define Logger interfaces in public packages that need logging
- [ ] Update all import paths across the codebase to use new package locations
- [ ] Add comprehensive package documentation (godoc) for all public packages
- [ ] Add usage examples in package documentation
- [ ] Ensure all public APIs have comprehensive unit test coverage
- [ ] Review public API surface for consistency and Go best practices

#### Step 13: Build System and Integration
- [ ] Complete Makefile with all targets
- [ ] Add comprehensive integration tests
- [ ] Cross-platform build support
- [ ] Final documentation and security review

#### Step 14: CI/CD Implementation
- [ ] Implement format, lint, test, and build pipeline for all commits
- [ ] Add pull request checks and status badges
- [ ] Implement SLSA compliant release pipeline

### Notes
- Each step should be completed with full TDD approach
- All tests must pass before moving to next step
- Regular integration testing throughout development
- Focus on incremental progress with working software at each step

### Development Principles
- Test-Driven Development (TDD) - write tests first
- Small, focused commits that build on each other
- No orphaned code - everything must integrate
- Comprehensive error handling and user feedback
- Follow Go best practices and conventions
