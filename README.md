# GitHub Project Import Extension

A  GitHub CLI extension that enables bulk import and migration of project items into GitHub Projects v2 from JSON or CSV files.

[![Go](https://github.com/mjeffryes/gh-project-import/actions/workflows/go.yml/badge.svg)](https://github.com/mjeffryes/gh-project-import/actions/workflows/go.yml)

## 🌟 Features

- **Bulk Import**: Import multiple project items at once from JSON or CSV files
- **All Item Types**: Support for draft issues, existing issues, and pull requests
- **Field Mapping**: Automatically map and convert field values including:
  - Text, Number, Date fields
  - Single-select dropdowns
  - User assignments
  - Iteration fields
- **Validation**: Comprehensive field validation against project schema
- **Dry Run**: Preview changes before execution
- **Progress Tracking**: Real-time import progress with detailed logging
- **Error Handling**: Robust error handling with clear error messages

## 📦 Installation

### Prerequisites

- [GitHub CLI](https://cli.github.com/) installed and authenticated
- Go 1.23.2 or later

### Install as GitHub CLI Extension

```bash
# Clone and build
git clone https://github.com/mjeffryes/gh-project-import.git
cd gh-project-import
make build

# Install as GitHub CLI extension
gh extension install .
```

### Manual Installation

```bash
# Build from source
git clone https://github.com/mjeffryes/gh-project-import.git
cd gh-project-import
make build

# Binary will be available as ./gh-project-import
```

## 🚀 Quick Start

### Basic Usage

```bash
# Import from JSON file
gh project-import --source items.json --project "owner/project-name"

# Import from CSV file
gh project-import --source items.csv --project "owner/project-name"

# Dry run to preview changes
gh project-import --source items.json --project "owner/project" --dry-run

# Verbose output
gh project-import --source items.json --project "owner/project" --verbose
```

### JSON Format Example

```json
[
  {
    "title": "Add user authentication",
    "notes": "Implement OAuth2 login flow",
    "Status": "Todo",
    "Priority": "High",
    "Estimate": 5,
    "Assignee": "octocat"
  },
  {
    "title": "Fix database connection issue",
    "url": "https://github.com/owner/repo/issues/123",
    "Status": "In Progress",
    "Sprint": "Sprint 3"
  }
]
```

### CSV Format Example

```csv
Title,URL,Status,Priority,Estimate,Notes,Sprint
Add user authentication,,Todo,High,5,Implement OAuth2 login flow,Sprint 3
Fix database issue,https://github.com/owner/repo/issues/123,In Progress,High,2,,Sprint 3
```

## 📖 Usage

### Command Line Options

| Option | Short | Description | Required |
|--------|-------|-------------|----------|
| `--source` | `-s` | Source file with items to import (JSON/CSV) | ✅ |
| `--project` | `-p` | Destination project identifier | ✅ |
| `--dry-run` | | Preview what would be imported without making changes | |
| `--verbose` | `-v` | Enable detailed logging | |
| `--quiet` | `-q` | Suppress non-error output | |

### Project Identifiers

The tool supports multiple project identifier formats:

- **Organization projects**: `org/project-name` (e.g., `github/Q4-Planning`)
- **User projects**: `username/project-name` (e.g., `octocat/Personal-Tasks`)
- **Project number**: `123` (numeric project ID)

### Field Mapping

The tool automatically maps fields from your input files to GitHub project fields:

#### Special Fields

- **`title`** (required): Item title
- **`url`**: GitHub issue/PR URL (creates linked items)
- **`notes`**: Item description/body (for draft issues)

#### Custom Fields

All other fields are mapped to project fields by name:

- **Text fields**: Any string value
- **Number fields**: Numeric values
- **Date fields**: ISO date format (YYYY-MM-DD)
- **Single-select fields**: Option names (case-sensitive)
- **User fields**: GitHub usernames
- **Iteration fields**: Iteration names

## 🏗️ Development

### Prerequisites

- Go 1.23.2+
- Make
- [gotestsum](https://github.com/gotestsum/gotestsum) (optional, for better test output)

### Setup

```bash
git clone https://github.com/mjeffryes/gh-project-import.git
cd gh-project-import
make install-gotestsum  # Optional: for better test output
```

### Common Commands

```bash
# Build the project
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Build for all platforms
make build-all

# Clean build artifacts
make clean

# Show all available targets
make help
```

### Project Structure

```
├── main.go              # CLI interface and import logic
├── github.go            # GitHub API client and operations
├── parser.go            # JSON/CSV parsing logic
├── snapshot.go          # Snapshot testing framework
├── fields_test.go       # Field conversion tests
├── integration_test.go  # End-to-end integration tests
├── testdata/           # Test fixtures and snapshots
└── Makefile            # Build and development tasks
```

## 🧪 Testing

The project includes comprehensive test coverage:

### Test Types

- **Unit Tests**: Core logic and field conversion
- **Integration Tests**: End-to-end workflow testing
- **Snapshot Tests**: GitHub API interaction testing

### Running Tests

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run with coverage report
make test-coverage

# Record new API snapshots (requires GitHub token)
SNAPSHOT_MODE=record make test-record-snapshots
```

### Snapshot Testing

The project uses a sophisticated snapshot testing system to test GitHub API interactions without making real API calls:

- **Default mode**: Replay from recorded snapshots
- **Record mode**: Make real API calls and record responses
- **Bypass mode**: Make real API calls without recording

See [SNAPSHOT_TESTING.md](SNAPSHOT_TESTING.md) for detailed information.

## 📊 Field Type Support

| Field Type | Input Format | Example |
|------------|--------------|---------|
| Text | String | `"Fix authentication bug"` |
| Number | Number or string | `5` or `"5"` |
| Date | ISO date | `"2024-12-31"` |
| Single Select | Option name | `"High"` |
| User | GitHub username | `"octocat"` |
| Iteration | Iteration name | `"Sprint 3"` |

## ⚠️ Important Notes

### Authentication

The tool uses your existing GitHub CLI authentication. Make sure you're authenticated and have appropriate permissions:

```bash
gh auth status
gh auth login  # if not authenticated
```

### Rate Limiting

The tool respects GitHub's rate limiting. For large imports, the process may take some time.

### Project Permissions

You need write access to the destination project to import items.

### Field Validation

- Fields not found in the destination project are skipped with warnings
- Invalid field values are logged but don't stop the import
- Use `--dry-run` to validate field mappings before importing

## 🤝 Contributing

We welcome contributions! Please see our contributing guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Development Workflow

1. Make your changes
2. Run `make check` to ensure tests pass and code is formatted
3. Add tests for new functionality
4. Update documentation as needed
5. Submit a pull request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built using [GitHub CLI Go library](https://github.com/cli/go-gh)
- Powered by [Cobra](https://github.com/spf13/cobra) for CLI interface
- Testing enhanced with [gotestsum](https://github.com/gotestsum/gotestsum)

## 📞 Support

- 🐛 **Bug Reports**: [Create an issue](https://github.com/mjeffryes/gh-project-import/issues)
- 💡 **Feature Requests**: [Create an issue](https://github.com/mjeffryes/gh-project-import/issues)
- ❓ **Questions**: [Discussions](https://github.com/mjeffryes/gh-project-import/discussions)

---

**Made with ❤️ for GitHub Projects v2**
