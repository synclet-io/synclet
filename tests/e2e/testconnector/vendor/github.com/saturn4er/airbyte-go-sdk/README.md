# Airbyte Go SDK

A Go SDK/CDK (Connector Development Kit) for building [Airbyte](https://airbyte.com) source connectors quickly and efficiently. This SDK abstracts away the complexities of the Airbyte protocol, allowing developers to focus on business logic rather than protocol implementation details.

## Features

- **Simple Interface**: Implement just 4 methods to create a fully functional Airbyte source connector
- **Strongly Typed**: Leverages Go's type system to help developers move fast without mistakes
- **Thread-Safe**: Built-in concurrency support with thread-safe message tracking
- **Protocol Compliant**: Implements Airbyte Protocol v0.5.2
- **Developer Friendly**: Focus on business logic while the SDK handles protocol details
- **Full Feature Support**:
  - Spec (connector configuration specification)
  - Check (connection validation)
  - Discover (schema discovery)
  - Read (data synchronization)
  - State management
  - Progress tracking with estimates
  - Structured logging

## Installation

```bash
go get github.com/saturn4er/airbyte-go-sdk
```

## Requirements

- Go 1.17 or higher
- [Task](https://taskfile.dev) (optional, for running examples and development tasks)

## Quick Start

### 1. Implement the Source Interface

Create a source by implementing the `airbyte.Source` interface:

```go
type Source interface {
    // Spec returns the input "form" spec needed for your source
    Spec(logTracker LogTracker) (*ConnectorSpecification, error)

    // Check verifies the source - usually verify creds/connection etc.
    Check(srcCfgPath string, logTracker LogTracker) error

    // Discover returns the schema of the data you want to sync
    Discover(srcConfigPath string, logTracker LogTracker) (*Catalog, error)

    // Read reads the actual data from your source and uses tracker methods
    // to sync data with Airbyte/destinations. MessageTracker is thread-safe,
    // so you can spin off goroutines to sync your data concurrently.
    Read(sourceCfgPath string, prevStatePath string, configuredCat *ConfiguredCatalog,
        tracker MessageTracker) error
}
```

### 2. Create Your Connector

```go
package main

import (
    "log"
    "os"

    "github.com/saturn4er/airbyte-go-sdk"
)

type MySource struct {
    // your source configuration
}

func (s MySource) Spec(logTracker airbyte.LogTracker) (*airbyte.ConnectorSpecification, error) {
    // Define your connector's configuration specification
    return &airbyte.ConnectorSpecification{
        DocumentationURL: "https://example.com/docs",
        ConnectionSpecification: airbyte.ConnectionSpecification{
            Title:       "My Source",
            Description: "My custom source connector",
            Type:        "object",
            Properties: airbyte.Properties{
                Properties: map[airbyte.PropertyName]airbyte.PropertySpec{
                    "api_key": {
                        Description: "API Key for authentication",
                        PropertyType: airbyte.PropertyType{
                            Type: []airbyte.PropType{airbyte.String},
                        },
                    },
                },
            },
            Required: []airbyte.PropertyName{"api_key"},
        },
    }, nil
}

func (s MySource) Check(srcCfgPath string, logTracker airbyte.LogTracker) error {
    // Validate your configuration and test the connection
    return nil
}

func (s MySource) Discover(srcConfigPath string, logTracker airbyte.LogTracker) (*airbyte.Catalog, error) {
    // Return the catalog of available streams
    return &airbyte.Catalog{
        Streams: []airbyte.Stream{
            {
                Name: "my_stream",
                JSONSchema: airbyte.Properties{
                    // Define your stream schema
                },
                SupportedSyncModes: []airbyte.SyncMode{
                    airbyte.SyncModeFullRefresh,
                },
            },
        },
    }, nil
}

func (s MySource) Read(sourceCfgPath string, prevStatePath string,
    configuredCat *airbyte.ConfiguredCatalog, tracker airbyte.MessageTracker) error {
    // Sync your data
    for _, stream := range configuredCat.Streams {
        // Fetch and emit records
        record := map[string]interface{}{
            "id": 1,
            "name": "example",
        }
        if err := tracker.Record(record, stream.Stream.Name, stream.Stream.Namespace); err != nil {
            return err
        }
    }
    return nil
}

func main() {
    mySource := MySource{}
    runner := airbyte.NewSourceRunner(mySource, os.Stdout)
    if err := runner.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### 3. Run Your Connector

```bash
# Build your connector
go build -o my-connector main.go

# Get connector specification
./my-connector spec

# Test connection
./my-connector check --config config.json

# Discover available streams
./my-connector discover --config config.json

# Sync data
./my-connector read --config config.json --catalog catalog.json
```

## Example: HTTP API Source

A complete example is included that demonstrates how to build a source connector for the DummyJSON API. See the [`examples/httpsource`](examples/httpsource) directory.

### Running the Example

Using Task (recommended):

```bash
# Run all commands in sequence
task demo

# Or run individual commands:
task spec      # Get connector specification
task check     # Validate configuration
task discover  # Get available streams
task read      # Sync data
```

Using Go directly:

```bash
# Build
cd examples/httpsource
go build -o ../../bin/httpsource main.go

# Run commands
../../bin/httpsource spec
../../bin/httpsource check --config config.json
../../bin/httpsource discover --config config.json
../../bin/httpsource read --config config.json --catalog catalog.json
```

## Core Concepts

### Source Interface

The `Source` interface is the only interface you need to implement:

- **Spec**: Define what configuration your connector needs (API keys, URLs, etc.)
- **Check**: Validate the provided configuration and test connectivity
- **Discover**: Return the schema of data streams available from your source
- **Read**: Fetch and emit records from your source

### Message Tracker

The `MessageTracker` interface provides thread-safe methods for emitting data during sync:

```go
type MessageTracker interface {
    // Record emits a data record
    Record(data interface{}, streamName string, namespace string) error

    // State emits state information for incremental syncs
    State(stateType StateType, stateData interface{}) error

    // Log emits a log message
    Log(level LogLevel, message string) error

    // Trace emits trace information (estimates, errors, etc.)
    Trace TraceEmitter
}
```

### Concurrency Support

The SDK is designed for concurrent data fetching:

```go
func (s MySource) Read(..., tracker airbyte.MessageTracker) error {
    var wg sync.WaitGroup

    for _, stream := range configuredCat.Streams {
        wg.Add(1)
        go func(stream ConfiguredStream) {
            defer wg.Done()
            // Fetch data concurrently
            for _, record := range fetchData(stream) {
                tracker.Record(record, stream.Stream.Name, stream.Stream.Namespace)
            }
        }(stream)
    }

    wg.Wait()
    return nil
}
```

## Development

### Available Tasks

```bash
task build         # Build the example connector
task test          # Run all tests
task test-short    # Run tests without verbose output
task fmt           # Format Go code
task lint          # Run golangci-lint
task dev           # Format, test, and build
task clean         # Clean build artifacts
```

### Running Tests

```bash
# Run all tests with verbose output
task test

# Or using go directly
go test -v ./...
```

## Project Structure

```
.
├── README.md              # This file
├── LICENSE               # MIT License
├── Taskfile.yml          # Task definitions for development
├── go.mod                # Go module definition
├── *.go                  # Core SDK implementation files
│   ├── source.go         # Source interface definition
│   ├── protocol.go       # Airbyte protocol types
│   ├── sourceRunner.go   # CLI runner implementation
│   ├── trackers.go       # Message tracking implementation
│   └── ...
├── schema/               # Schema utilities
│   └── schema.go
└── examples/             # Example connectors
    └── httpsource/       # HTTP API source example
        ├── main.go       # Entry point
        ├── config.json   # Example configuration
        ├── catalog.json  # Example catalog
        └── apisource/    # Source implementation
            └── apisource.go
```

## Testing Your Connector

1. **Test Spec**: Ensure your spec returns valid configuration schema
2. **Test Check**: Verify connection validation works correctly
3. **Test Discover**: Confirm schema discovery returns expected streams
4. **Test Read**: Validate that data is synced correctly with proper formatting

```go
func TestMySource(t *testing.T) {
    source := MySource{}

    // Test spec
    spec, err := source.Spec(mockLogTracker)
    assert.NoError(t, err)
    assert.NotNil(t, spec)

    // Test check
    err = source.Check("config.json", mockLogTracker)
    assert.NoError(t, err)

    // Test discover
    catalog, err := source.Discover("config.json", mockLogTracker)
    assert.NoError(t, err)
    assert.Greater(t, len(catalog.Streams), 0)
}
```

## Advanced Features

### State Management

Support incremental syncs by maintaining state:

```go
func (s MySource) Read(...) error {
    // Load previous state
    var state MyState
    airbyte.UnmarshalFromPath(prevStatePath, &state)

    // Fetch data since last sync
    records := fetchDataSince(state.LastTimestamp)

    for _, record := range records {
        tracker.Record(record, streamName, namespace)
    }

    // Save new state
    return tracker.State(airbyte.StateTypeLegacy, MyState{
        LastTimestamp: time.Now().Unix(),
    })
}
```

### Progress Tracking

Emit estimates for better sync visibility:

```go
rowEstimate := int64(1000)
airbyte.EmitEstimate(
    tracker.Trace,
    streamName,
    &namespace,
    airbyte.EstimateTypeStream,
    &rowEstimate,
    nil,
)
```

## Contributing

Contributions are welcome! Please feel free to submit issues, fork the repository, and create pull requests.

### Development Setup

1. Clone the repository
2. Install dependencies: `go mod download`
3. Install Task: `brew install go-task/tap/go-task` (macOS) or see [Task installation](https://taskfile.dev/installation/)
4. Run tests: `task test`
5. Build example: `task build`

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Resources

- [Airbyte Documentation](https://docs.airbyte.com)
- [Airbyte Protocol](https://docs.airbyte.com/understanding-airbyte/airbyte-protocol)
- [Building Connectors](https://docs.airbyte.com/connector-development)
- [Go Documentation](https://golang.org/doc/)

## Support

For questions, issues, or contributions, please open an issue on GitHub.

---

Built with by [@saturn4er](https://github.com/saturn4er)