# GitHub Project Import Extension Specification

## Overview

The `gh-project-import` extension enables users to import project items from a json or csv file into a GitHub Projects v2 board.
This tool is designed to help teams automate bulk additions, synchronize, or migrate between projects.

## Core Functionality

### Primary Use Cases
- [x] **Bulk Additions**: Add multiple items to a project at once
- [x] **Project Migration**: Move all items from an old project (eg. from an export) to a new project structure

### Key Features
- [x] Bulk import multiple items (all item types: PR, issue, draft)
- [x] Set field values for the newly added items
- [x] Validate input against the field types and allowed values
- [x] Support for all GitHub Projects v2 field types (text, number, date, single-select, iteration, etc.)
- [x] Dry-run mode to preview changes before execution
- [x] Detailed logging and progress reporting

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
- [x] Add items from json source file to destination project
    Example input
    ``` json
    [
        {
          "assignees": [
            "mjeffryes"
          ],
          "content": {
            "body": "",
            "title": "Company Holiday",
            "type": "DraftIssue"
          },
          "estimate (Days)": 1,
          "id": "PVTI_lADOAU-UG84AZbl-zgamtPQ",
          "m": {
            "duration": 21,
            "startDate": "2025-05-19",
            "title": "M121"
          },
          "title": "Company Holiday"
        },
        {
          "content": {
            "body": "Extend this clean-up job to also clean up big table resources. ",
            "number": 31,
            "repository": "gizmoduck/gcp-account-cleanup",
            "title": "Cleanup big table",
            "type": "Issue",
            "url": "https://github.com/gizmoduck/gcp-account-cleanup/issues/31"
          },
          "estimate (Days)": 1,
          "id": "PVTI_lADOAU-UG84AZbl-zgb8xzc",
          "labels": [
            "kind/enhancement",
            "awaiting-feedback"
          ],
          "notes": "Cost optimization?",
          "repository": "https://github.com/gizmoduck/gcp-account-cleanup",
          "status": "Held",
          "title": "Cleanup big table"
        },
    ]
    ```
- [x] Add items from csv source file to destination project
    Example input
    ``` csv
    Repository,Title,URL,Assignees,Estimate (Days),Spent (Days),Theme,Status,notes,M
    gizmoduck/gizmoduck-cloudy,Unable to ignore target weights on an ALB with a wildcard expression,https://github.com/gizmoduck/gizmoduck-cloudy/issues/5771,mjeffryes,2,,orange bugs,Proposed,,M125
    ,Ops 8/25 - 8/29,,octocat,5,,Provider ops,,,M125
    gizmoduck/gizmoduck-nimbus,GCP v9.0: release alpha version,https://github.com/gizmoduck/gizmoduck-nimbus/issues/3312,octocat,3,,,Proposed,,M125
    gizmoduck/gizmoduck-nimbus,GCP v9.0: test alpha version against examples,https://github.com/gizmoduck/gizmoduck-nimbus/issues/3319,octocat,4,,,Proposed,,M125
    ```

### Required Parameters
- [x] `--source, -s`: Source file with items to import
- [x] `--project, -p`: Destination project identifier (format: `owner/project-name` or `project-number`)

### Optional Parameters
- [x] `--dry-run`: Preview what would be copied without making changes
- [x] `--verbose, -v`: Enable verbose logging
- [x] `--quiet, -q`: Suppress non-error output

### Project Identifier Formats
- [x] **Organization projects**: `org/project-name` (e.g., `github/Q4 Planning`)
- [x] **User projects**: `username/project-name` (e.g., `octocat/Personal Tasks`)
- [x] **Numbered format**: `owner/123` (e.g., `octocat/123`)

## Internal structure

- [x] main.go - parsing user input and core import logic
    - [x] parse CLI flags and arguments
    - [x] discover fields on destination project
    - [x] validate fields and values in the source file match the field schema of the destination project
    - [x] add items from the file to the project (including setting field values)
- [x] github.go - library functions that wrap API calls to GitHub
    - [x] **Project Discovery**: Query projects by name or number
    - [x] **Field Schema**: Retrieve field definitions and options
    - [x] **Item Creation**: Add items to destination project
    - [x] **Field Updates**: Set field values on copied items
    - [x] **Create/Delete Project**: Create or delete a project (only used in e2e test)

IMPORTANT: All interactions with the github API should be mediated via the functions in github.go

## Input formats and field mapping

Ignored Fields:
- Id (maybe present from project exports)
- Repository (depend on the url field instead)
- Labels (already set on the issue/PR)
- Milestone (already set on the issue/PR)

Required Fields:
- Title

Optional
- Url (empty/missing url indicates a draft item)
- Assignee (only used when importing draft issues)
- *all other field names are assumed to be the name of fields on the target project

### Field Type Compatibility Matrix
Input validation for destination project field types
- [x] Text: input value converted to a string
- [x] Number: input must be a number or a string containing only a number
- [x] Date: input must be in ISO Date format
- [x] Single Select: input must be the name of one of the single-select values
- [x] User: input must be user login as text
- [x] Iteration: input must be iteration name as text


## Output and Logging

### Success Output

- [x] Basic success/failure reporting
- [ ] Field mapping preservation count
- [ ] Skipped fields due to compatibility issues
- [ ] Detailed field compatibility warnings

Example:
```
✓ Copied 15 items from "Sprint Planning" to "Q4 Sprint"
✓ Preserved 8 field mappings
⚠ Skipped 2 fields due to compatibility issues
   - "Custom Priority" field not found in destination
   - "Team Assignment" field type mismatch
```

### Verbose Logging

- [x] Authentication status
- [x] Project resolution logging
- [x] Item retrieval progress
- [ ] Field compatibility analysis logging
- [ ] Individual item copy progress
- [ ] Field value setting details

```
[INFO] Authenticating with GitHub API...
[INFO] Resolving source project: octocat/repo/Sprint Planning
[INFO] Found project ID: PVT_kwDOABCD123
[INFO] Resolving destination project: octocat/repo/Q4 Sprint
[INFO] Found project ID: PVT_kwDOEFGH456
[INFO] Retrieving source project items... (15 items found)
[INFO] Analyzing field compatibility...
[INFO] Creating field mapping: Priority -> Importance (Single Select)
[WARN] Skipping field: Custom Field (not found in destination)
[INFO] Copying item 1/15: "Implement user authentication"
[INFO] Setting field values: Status=In Progress, Priority=High, Assignee=@octocat
[SUCCESS] Item copied successfully
...
```

## Implementation Details

### Performance Considerations
- [ ] **Batch operations**: Group API calls where possible to minimize requests
- [x] **Caching**: Cache project schemas and field definitions
- [x] **Progress tracking**: Show progress for large copy operations

### Testing Strategy
- [x] **Unit tests**:
    - [x] field mapping logic
    - [x] interpretation of the user inputs
    - [x] Error handling
- [x] **Snapshot tests**:
    - [x] Record/replay github API interactions for all the functions in github.go
    - [x] Default should be to replay from the snapshot.
    - [x] Make real API calls and record a new snapshot when an environment variable is set
- [x] **Integration tests**: Test with real GitHub projects (using test repositories)
    - [x] Create a new source and destination project in this repository
    - [x] Add test data to the source project
    - [x] Test copying items from the source project to the destination
    - [x] Cleanup the test projects when done

## Security Considerations
- [x] **Token handling**: Secure handling of GitHub authentication tokens
