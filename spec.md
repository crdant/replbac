# replbac - Replicated RBAC Synchronization Tool

## Overview
`replbac` is a Go-based command-line tool that synchronizes RBAC (Role-Based Access Control) configurations between local YAML files and the Replicated Vendor Portal API.

## Core Functionality

### File Structure
- **Input Format**: Individual YAML files, one per role
- **Directory Structure**: Full tree structure supported (roles can be organized in subdirectories)
- **YAML Structure**: Simplified format without the API's `v1` wrapper
```yaml
name: "View Customers Only"
resources:
  allowed:
    - "kots/app/*/license/*/read"
    - "kots/app/*/license/*/list"
  denied:
    - "**/*"
```

### Commands
- **Primary Command**: Sync roles from local files to Vendor Portal
- **Subcommands**:
  - `init`/`bootstrap`: Pull existing roles from Vendor Portal and generate local YAML files
  - Additional subcommands as appropriate for functionality
- **Flags**:
  - `--dry-run`: Show changes without applying them (boolean flag)
  - `--confirm` / `--no-confirm`: Control confirmation prompts (default: confirm enabled)

## Authentication & Configuration

### API Token Management
Support multiple methods:
- Environment variables
- Command-line flags  
- Configuration files

### Configuration File
- **Format**: YAML
- **Contents**: Default settings including:
  - API endpoints
  - Token sources
  - Confirmation flag defaults
  - Logging preferences

### Environment Variables
- API token configuration
- Confirmation state override (`--confirm`/`--no-confirm` flag state)

## Behavior & Error Handling

### Conflict Resolution
- **Default**: Prompt for confirmation before overwriting existing roles
- **Override**: `--confirm`/`--no-confirm` flags control prompting behavior
- **Environment Variable**: Available to set default confirmation state

### Role Mapping
- **Name Source**: Use the `name` field within each YAML file
- **File Processing**: Each role is independent (no cross-references or dependencies)

### Error Handling
- **Invalid YAML/API Validation Failures**: Log errors and prompt user to continue or stop
- **Missing Portal Roles**: Leave existing portal roles without local files unchanged
- **Processing Order**: Files processed independently (no dependency management needed)

### Dry Run Mode
- Show what changes would be made without applying them
- Available via `--dry-run` flag

## Technical Implementation

### Architecture
- **Language**: Go
- **Project Structure**: Follow [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- **CLI Framework**: Cobra (common Go CLI framework)
- **Build System**: Make for local builds
- **CI/CD**: GitHub Actions

### Logging
- **Framework**: Current Go community convention
- **Levels**: Standard log levels (debug, info, warn, error)
- **Format**: Standard Go logging practices

### API Integration
- **Validation**: Server-side validation (rely on Vendor Portal API)
- **Target API**: Replicated Vendor Portal RBAC API
- **JSON Format**: Transform YAML to required API format with `v1` wrapper

## Future Enhancements
- File watching and automatic synchronization (deferred)
- Additional sync strategies and conflict resolution options

## Success Criteria
- Seamless bidirectional sync between local YAML files and Vendor Portal
- Intuitive CLI interface following Go conventions
- Robust error handling and user feedback
- Support for various authentication and configuration methods
- Dry-run capability for safe testing