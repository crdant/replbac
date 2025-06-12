# replbac - Replicated RBAC Synchronization Tool 🔒

`replbac` is a command-line tool for synchronizing Role-Based Access Control (RBAC) configurations between local YAML files and the Replicated Vendor Portal API. Manage your Replicated team roles as code with version control and automated deployment.

## 🌟 Features

- **Bidirectional Synchronization**: Push local role definitions to the Replicated platform or pull down existing roles
- **Simple YAML Format**: Manage roles in easy-to-read YAML files
- **Dry-Run Mode**: Preview changes before applying them
- **Flexible Configuration**: Multiple ways to configure API credentials
- **Comprehensive Error Handling**: Clear, actionable error messages
- **Deletion Control**: Safely control which roles get deleted with the `--delete` flag

## 📋 Requirements

- Go 1.16 or later (for building from source)
- A Replicated Vendor account with API token

## 🔧 Installation

### From Binary Release

Download the latest release for your platform from the [Releases](https://github.com/crdant/replbac/releases) page.

```bash
# Make it executable (Linux/macOS)
chmod +x replbac

# Move to a directory in your PATH
sudo mv replbac /usr/local/bin/
```

### Using Go

```bash
go install github.com/crdant/replbac/cmd/replbac@latest
```

### From Source

```bash
# Clone the repository
git clone https://github.com/crdant/replbac.git
cd replbac

# Build the binary
make build

# Install it to your GOPATH
make install
```

## ⚙️ Configuration

`replbac` supports multiple configuration methods with the following precedence (highest to lowest):

1. Command-line flags
2. Environment variables
3. Default values

### API Token

Set your Replicated API token using one of these methods:

```bash
# Environment variable (recommended)
export REPLICATED_API_TOKEN=your-api-token

# Or use the REPLBAC-specific variable
export REPLBAC_API_TOKEN=your-api-token

# Or use the command-line flag
replbac --api-token=your-api-token
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `REPLICATED_API_TOKEN` | Replicated API token (preferred) |
| `REPLBAC_API_TOKEN` | Alternative API token source |
| `REPLBAC_LOG_LEVEL` | Log level (debug, info, warn, error) |
| `REPLBAC_CONFIRM` | Auto-confirm operations (true/false) |
| `REPLBAC_CONFIG` | Path to config file |

## 🚀 Usage

### Synchronize Local Roles to Replicated (Push)

```bash
# Sync roles from current directory
replbac sync

# Sync roles from a specific directory
replbac sync /path/to/roles

# Preview changes without applying them
replbac sync --dry-run

# Enable verbose logging
replbac sync --verbose

# Delete remote roles not present in local files
replbac sync --delete

# Skip confirmation prompts for deletions
replbac sync --delete --force
```

### Download Roles from Replicated to Local Files (Pull)

```bash
# Pull roles to current directory
replbac pull

# Pull roles to a specific directory
replbac pull /path/to/roles

# Preview without making changes
replbac pull --dry-run

# Overwrite existing files
replbac pull --force

# Show detailed differences 
replbac pull --diff
```

### Role File Format

Create one YAML file per role:

```yaml
# admin.yaml
name: admin
resources:
  allowed:
    - "**/*"
  denied:
    - "kots/app/*/delete"
```

```yaml
# viewer.yaml
name: viewer
resources:
  allowed:
    - "kots/app/*/read"
    - "team/support-issues/read"
  denied:
    - "kots/app/*/write"
    - "kots/app/*/delete"
    - "kots/app/*/admin"
```

### Show Version Information

```bash
replbac version
```

## 🧩 Commands

| Command | Description |
|---------|-------------|
| `sync` | Synchronize local role files to Replicated API |
| `pull` | Download remote roles to local YAML files |
| `version` | Display version information |
| `help` | Display help information for any command |

## 🔄 Sync Command Options

| Option | Description |
|--------|-------------|
| `--dry-run` | Preview changes without applying them |
| `--diff` | Show detailed differences (implies --dry-run) |
| `--delete` | Delete remote roles not present in local files |
| `--force` | Skip confirmation prompts (requires --delete) |
| `--verbose` | Enable info-level logging |
| `--debug` | Enable debug-level logging |

## 🚦 Pull Command Options

| Option | Description |
|--------|-------------|
| `--dry-run` | Preview changes without applying them |
| `--diff` | Show detailed differences (implies --dry-run) |
| `--force` | Overwrite existing files |
| `--verbose` | Enable info-level logging |
| `--debug` | Enable debug-level logging |

## 🔍 Global Options

| Option | Description |
|--------|-------------|
| `--api-token` | Replicated API token |
| `--config` | Path to config file |
| `--log-level` | Log level (debug, info, warn, error) |
| `--confirm` | Auto-confirm destructive operations |

## 🛠️ Deployment Workflows

### Continuous Integration

Integrate `replbac` into your CI/CD pipeline to automatically sync role changes:

```yaml
# Example GitHub Actions workflow
name: Sync RBAC Roles

on:
  push:
    branches: [ main ]
    paths:
      - 'roles/**'

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Setup Go
        uses: actions/setup-go@v2
        with:
          go-version: '1.16'

      - name: Install replbac
        run: go install github.com/crdant/replbac/cmd/replbac@latest

      - name: Sync roles
        run: replbac sync roles --delete --force
        env:
          REPLICATED_API_TOKEN: ${{ secrets.REPLICATED_API_TOKEN }}
```

### Development Workflow

A typical development workflow:

1. Pull existing roles: `replbac pull roles`
2. Make changes to role files
3. Test changes: `replbac sync --dry-run`
4. Apply changes: `replbac sync`
5. Commit and push changes

## 🔐 Security Considerations

- Store your API token securely and never commit it to version control
- Use environment variables for API tokens in scripts and CI/CD workflows
- The `--force` flag skips confirmation prompts - use with caution
- When using `--delete`, consider first running with `--dry-run` to preview deletions

## ⚠️ Error Handling

`replbac` provides clear error messages and recovery suggestions:

- **Configuration errors**: Check your API token
- **File errors**: Ensures YAML files are properly formatted
- **Network errors**: Retries for transient failures with clear messages
- **Validation errors**: Specific guidance on role validation issues

## 🧪 Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/crdant/replbac.git
cd replbac

# Run tests
make test

# Build for your platform
make build

# Build for all platforms
make build-all
```

### Running Tests

```bash
# Run all tests
make test

# Run specific tests
go test ./internal/api -v
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Commit your changes: `git commit -am 'Add new feature'`
4. Push to the branch: `git push origin feature/my-feature`
5. Submit a pull request

## 📜 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Built for the Replicated community
- Inspired by infrastructure-as-code practices
