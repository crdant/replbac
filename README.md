# replbac - Replicated RBAC Synchronization Tool 🔒

[![Go Report Card](https://goreportcard.com/badge/github.com/crdant/replbac)](https://goreportcard.com/report/github.com/crdant/replbac)

`replbac` is a command-line tool for synchronizing Role-Based Access Control (RBAC) configurations between local YAML files and the Replicated Vendor Portal API. Manage your Replicated team roles as code with version control and automated deployment.

## 🌟 Features

- **Bidirectional Sync**: Push local role definitions to the Replicated platform or pull down existing roles
- **Simple YAML Format**: Manage roles in easy-to-read YAML files
- **Dry-Run Mode**: Preview changes before applying them
- **Flexible Configuration**: Multiple ways to configure API credentials
- **Comprehensive Error Handling**: Clear, actionable error messages
- **Verbose Logging**: Detailed logging for troubleshooting

## 📋 Requirements

- Go 1.16 or later
- A Replicated Vendor account with API token

## 🔧 Installation

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

### Download Binary

Download the latest release for your platform from the [Releases](https://github.com/crdant/replbac/releases) page.

## ⚙️ Configuration

`replbac` supports multiple configuration methods with the following precedence (highest to lowest):

1. Command-line flags
2. Environment variables
3. Configuration file
4. Default values

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

### Configuration File

Create a configuration file at one of these locations:

- macOS: `~/Library/Preferences/com.replicated.replbac/config.yaml`
- Linux: `~/.config/replbac/config.yaml`
- Windows: `~/.replbac/config.yaml`

Or specify a custom path:

```bash
replbac --config=/path/to/config.yaml
```

Example configuration file:

```yaml
api_endpoint: https://api.replicated.com
api_token: your-api-token-here
log_level: info
confirm: true
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `REPLICATED_API_TOKEN` | Replicated API token (preferred) |
| `REPLBAC_API_TOKEN` | Alternative API token source |
| `REPLBAC_API_ENDPOINT` | API endpoint URL |
| `REPLBAC_LOG_LEVEL` | Log level (debug, info, warn, error) |
| `REPLBAC_CONFIRM` | Auto-confirm operations (true/false) |
| `REPLBAC_CONFIG` | Path to config file |

## 🚀 Usage

### Synchronize Local Roles to Replicated

```bash
# Sync roles from current directory
replbac sync

# Sync roles from a specific directory
replbac sync /path/to/roles

# Preview changes without applying them
replbac sync --dry-run

# Use a specific roles directory
replbac sync --roles-dir=/path/to/roles

# Enable verbose logging
replbac sync --verbose
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
| `init` | Initialize local role files from existing Replicated API roles |
| `version` | Display version information |
| `help` | Display help information for any command |

## 🔄 Sync Command Options

| Option | Description |
|--------|-------------|
| `--dry-run` | Preview changes without applying them |
| `--roles-dir` | Directory containing role YAML files |
| `--verbose` | Enable verbose logging |

## 🚦 Init Command Options

| Option | Description |
|--------|-------------|
| `--output-dir` | Directory to create role files |
| `--force` | Overwrite existing files |

## 🔍 Global Options

| Option | Description |
|--------|-------------|
| `--api-token` | Replicated API token |
| `--api-endpoint` | Replicated API endpoint URL |
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
        run: replbac sync roles
        env:
          REPLICATED_API_TOKEN: ${{ secrets.REPLICATED_API_TOKEN }}
```

### Development Workflow

A typical development workflow:

1. Clone the roles repository
2. Make changes to role files
3. Test changes with `replbac sync --dry-run`
4. Apply changes with `replbac sync`
5. Commit and push changes

## ⚠️ Error Handling

`replbac` provides clear error messages and recovery suggestions:

- **Configuration errors**: Check your API token and endpoints
- **File errors**: Ensures YAML files are properly formatted
- **Network errors**: Retries for transient failures with clear messages
- **Validation errors**: Specific guidance on role validation issues

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📜 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Built for the Replicated community
- Inspired by infrastructure-as-code practices