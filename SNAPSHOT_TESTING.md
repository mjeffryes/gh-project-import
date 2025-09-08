# Snapshot Testing

This project includes comprehensive snapshot testing for GitHub API interactions, allowing deterministic tests without making real API calls.

## How Snapshot Tests Work

Snapshot tests record GitHub API interactions and replay them during test execution. This provides:

- **Deterministic testing**: Tests always produce the same results
- **Fast execution**: No network calls during test runs
- **Offline development**: Tests work without internet connectivity
- **API stability**: Tests aren't affected by GitHub API changes or rate limits

## Modes of Operation

### Replay Mode (Default)
```bash
# Runs tests using recorded snapshots
go test -v -run TestSnapshot

# Explicitly set replay mode
SNAPSHOT_MODE=replay go test -v -run TestSnapshot
```

### Record Mode
```bash
# Records new snapshots from real API calls
SNAPSHOT_MODE=record go test -v -run TestSnapshot
```

**⚠️ Warning**: Record mode makes real GitHub API calls and requires valid authentication.

### Bypass Mode
```bash
# Makes real API calls without recording (for debugging)
SNAPSHOT_MODE=bypass go test -v -run TestSnapshot
```

## Environment Variables

- `SNAPSHOT_MODE`: Controls test mode (`replay` | `record` | `bypass`)
- `SNAPSHOT_DIR`: Directory for snapshot files (default: `testdata/snapshots`)

## Snapshot File Structure

Snapshots are stored as JSON files in `testdata/snapshots/`:

```json
{
  "test_name": "GetUser",
  "calls": [
    {
      "method": "GET",
      "url": "user",
      "status_code": 200,
      "response": "{\"login\":\"test-user\",\"id\":12345}",
      "timestamp": "2024-01-01T10:00:00Z"
    }
  ],
  "created": "2024-01-01T10:00:00Z",
  "updated": "2024-01-01T10:00:00Z"
}
```

## Available Snapshot Tests

| Test Function | Description |
|---------------|-------------|
| `TestSnapshotGetUser` | User authentication |
| `TestSnapshotFindProject` | Project discovery by ID/name |
| `TestSnapshotGetProjectFields` | Field schema retrieval |
| `TestSnapshotCreateDraftIssue` | Draft issue creation |
| `TestSnapshotCreateProjectItem` | Adding existing items to projects |
| `TestSnapshotGetIssueOrPR` | Issue/PR content retrieval |
| `TestSnapshotSetProjectItemFieldValue` | Field value updates |
| `TestSnapshotEndToEndWorkflow` | Complete import workflow |

## Recording New Snapshots

When recording new snapshots, ensure you have:

1. Valid GitHub authentication (gh CLI logged in)
2. Access to test repositories and projects
3. Appropriate permissions for the operations being tested

```bash
# Record snapshots for specific tests
SNAPSHOT_MODE=record go test -v -run TestSnapshotGetUser

# Record all snapshots
SNAPSHOT_MODE=record go test -v -run TestSnapshot
```

## Best Practices

1. **Use meaningful test names**: Test names become snapshot filenames
2. **Keep snapshots minimal**: Only record necessary API calls
3. **Version control snapshots**: Include snapshot files in git
4. **Update when APIs change**: Re-record when GitHub API responses change
5. **Validate recorded data**: Ensure snapshots contain realistic test data

## Troubleshooting

### Missing Snapshot Files
```
Error: failed to read snapshot file ... (try running with SNAPSHOT_MODE=record to create it)
```
**Solution**: Run the test with `SNAPSHOT_MODE=record` to create the snapshot.

### API Call Mismatch
```
Error: method mismatch: expected GET, got POST
```
**Solution**: The test is making different API calls than recorded. Re-record the snapshot.

### Authentication Errors (Record Mode)
```
Error: failed to create GitHub client
```
**Solution**: Ensure GitHub CLI is authenticated with `gh auth status`.

## Implementation Details

The snapshot testing system consists of:

- `SnapshotGitHubClient`: Wrapper client that records/replays API calls
- `snapshot.go`: Core snapshot recording and replay logic
- `snapshot_test.go`: Test functions using snapshot client
- `testdata/snapshots/`: Directory containing recorded API interactions

This system allows comprehensive testing of all GitHub API interactions without requiring real API access during normal test execution.

## Real Integration Tests

In addition to snapshot tests, this project includes real integration tests that create actual GitHub projects to test the full workflow. These tests are located in `real_integration_test.go` and use build tags to prevent them from running during normal test execution.

### Running Real Integration Tests

Real integration tests require:

1. **GitHub authentication**: Valid GitHub CLI authentication (`gh auth status`)
2. **Environment variable**: Set `REAL_INTEGRATION_TESTS=true` 
3. **Build tag**: Include `-tags=integration`

```bash
# Run real integration tests
REAL_INTEGRATION_TESTS=true go test -v -tags=integration -run TestReal

# Run a specific real integration test
REAL_INTEGRATION_TESTS=true go test -v -tags=integration -run TestRealIntegrationWorkflow
```

**⚠️ Warning**: Real integration tests create and delete actual GitHub projects in your account. They include cleanup logic but should be used with caution.

### Available Real Integration Tests

| Test Function | Description |
|---------------|-------------|
| `TestRealIntegrationWorkflow` | Complete end-to-end workflow with project creation, data import, and cleanup |
| `TestRealProjectCreationAndDeletion` | Tests basic project lifecycle operations |
| `TestRealErrorHandling` | Tests error scenarios with real API calls |

### Test Behavior

Real integration tests:
- Create temporary projects with timestamped names to avoid conflicts
- Test actual API interactions including project creation/deletion
- Include comprehensive cleanup in defer functions
- Skip automatically if `REAL_INTEGRATION_TESTS` is not set to `true`
- Validate all operations against real GitHub API responses