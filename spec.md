# GitHub Project Import Extension Specification

## Overview

The `gh-project-import` extension enables users to import project items from a json, csv or tsv file into a GitHub Projects v2 board.
This tool is designed to help teams automate bulk additions, synchronize, or migrate between projects.

## Core Functionality

### Primary Use Cases
- [ ] **Bulk Additions**: Add multiple items to a project at once
- [ ] **Project Migration**: Move all items from an old project (eg. from an export) to a new project structure

### Key Features
- [ ] Bulk import multiple items (all item types: PR, issue, draft)
- [ ] Set field values for the newly added items
- [ ] Validate input against the field types and allowed values
- [ ] Support for all GitHub Projects v2 field types (text, number, date, single-select, iteration, etc.)
- [ ] Dry-run mode to preview changes before execution
- [ ] Detailed logging and progress reporting

## Command-Line Interface

```bash
# import from a json file
gh project-import --source "import_file.json" --project "owner/project"
```

```bash
# import a csv file
gh project-import --source "import_file.csv" --project "owner/project"
```

### Basic Usage Commands
- [ ] Add items from json source file to destination project
    Example input
    ``` json
    ```
- [ ] Add items from csv source file to destination project
    Example input
    ``` csv
    ```

### Required Parameters
- [ ] `--source, -s`: Source file with items to import
- [ ] `--project, -p`: Destination project identifier (format: `owner/project-name` or `project-number`)

### Optional Parameters
- [ ] `--dry-run`: Preview what would be copied without making changes
- [ ] `--verbose, -v`: Enable verbose logging
- [ ] `--quiet, -q`: Suppress non-error output

### Project Identifier Formats
- [ ] **Organization projects**: `org/project-name` (e.g., `github/Q4 Planning`)
- [ ] **User projects**: `username/project-name` (e.g., `octocat/Personal Tasks`)
- [ ] **Numbered format**: `owner/123` (e.g., `octocat/123`)

## Internal structure

- [ ] main.go - parsing user input and core import logic
    - [ ] parse CLI flags and arguments
    - [ ] discover fields on destination project
    - [ ] validate fields and values in the source file match the field schema of the destination project
    - [ ] add items from the file to the project (including setting field values)
- [ ] github.go - library functions that wrap API calls to GitHub
    - [ ] **Project Discovery**: Query projects by name or number
    - [ ] **Field Schema**: Retrieve field definitions and options
    - [ ] **Item Creation**: Add items to destination project
    - [ ] **Field Updates**: Set field values on copied items
    - [ ] **Create/Delete Project**: Create or delete a project (only used in e2e test)

IMPORTANT: All interactions with the github API should be mediated via the functions in github.go

## Error Handling and Edge Cases

### Common Error Scenarios

1. **Permission Errors**
   - [ ] Insufficient permissions on destination project
   - [ ] Project not accessible to authenticated user
   - [ ] Repository or organization access denied

2. **Field Compatibility Errors**
   - [ ] Destination field doesn't exist
   - [ ] Field types incompatible (e.g., trying to copy text to number field)
   - [ ] Invalid field values (e.g., non-existent single-select option)

3. **Item Reference Errors**
   - [ ] Item already exists in destination project (Warn only)
   - [ ] Referenced issues/PRs not accessible

4. **Rate Limiting**
   - [ ] GitHub API rate limit exceeded
   - [ ] Automatic retry with exponential backoff

### Error Recovery
- [ ] **Partial failure handling**: Continue with other items when individual items fail
- [ ] **Resume functionality**: Can safely retry an interrupted import
- [ ] **Detailed error reporting**: Log specific failures with actionable error messages

## Output and Logging

### Success Output

- [ ] Basic success/failure reporting
- [ ] Detailed field compatibility warnings

Example:
```
✓ Imported 15 items to "Q4 Sprint"
```

### Error/Warning outputs
 - [ ] Warning on item already exists

Example:
```
[WARN] Skipping https://github.com/org/repo/issue/123 already exists in project "Q4 Sprint"
✓ Imported 15 items to "Q4 Sprint"
⚠ Skipped 1 pre existing item
```

### Verbose Logging

- [ ] Authentication status
- [ ] Project resolution logging
- [ ] Individual item import progress
- [ ] Field value setting details

```
[INFO] Authenticating with GitHub API...
[INFO] Resolving destination project: octocat/repo/Q4 Sprint
[INFO] Found project ID: PVT_kwDOEFGH456
[INFO] Retrieving source items... (15 items found)
[INFO] Analyzing field compatibility...
[INFO] Importing item 1/15: "Implement user authentication" (Draft)
[INFO] Setting field values: Status=In Progress, Priority=High, Assignee=@octocat
[SUCCESS] Item copied successfully
[INFO] Importing item 2/15: "Transfer widgets" (https://github.com/org/repo/issue/123)
[WARN] Skipping https://github.com/org/repo/issue/123 already exists in project "Q4 Sprint"
[SUCCESS] Item copied successfully
[INFO] Importing item 3/15: "Integrate libraries" (https://github.com/org/repo/issue/124)
[INFO] Setting field values: Status=Blocked, Priority=High, Assignee=@octocat
[SUCCESS] Item copied successfully
...
```

## Implementation Details

### Performance Considerations
- [ ] **Batch operations**: Group API calls where possible to minimize requests
- [ ] **Caching**: Cache project schemas and field definitions
- [ ] **Progress tracking**: Show progress for large copy operations

### Testing Strategy
- [ ] **Unit tests**:
    - [ ] Interpretation of the user inputs/flags
    - [ ] Parsing of json format
    - [ ] Parsing of csv format
    - [ ] Error handling
- [ ] **Snapshot tests**:
    - [ ] Record/replay github API interactions for all the functions in github.go
    - [ ] Default should be to replay from the snapshot.
    - [ ] Make real API calls and record a new snapshot when an environment variable is set
- [ ] **Integration tests**: Test with real GitHub projects (using test repositories)
    - [ ] Create a new destination project in this repository
    - [ ] Test importing items from a source file to the destination
    - [ ] Cleanup the test project when done

## Security Considerations
- [ ] **Token handling**: Secure handling of GitHub authentication tokens
