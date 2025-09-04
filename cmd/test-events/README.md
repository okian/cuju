# Cuju Event Test Tool

A high-performance concurrent Go CLI tool for testing the Cuju event processing system. This tool ports the functionality of the original `test_1000_events.sh` shell script to Go with significant performance improvements through concurrent processing.

## Features

- **High Performance**: Concurrent event generation, submission, and ranking retrieval
- **Configurable Concurrency**: Adjustable worker pools for optimal performance
- **Real-time Progress**: Live progress reporting during execution
- **Comprehensive Statistics**: Detailed performance metrics and success rates
- **Result Verification**: Automatic verification of rankings and leaderboard consistency
- **Event Persistence**: Saves generated events to JSON files for analysis
- **Flexible Configuration**: Command-line options for all parameters
- **File Logging**: All output is logged to files for analysis and debugging

## Performance Improvements

Compared to the original shell script, this Go implementation provides:

- **~200x faster event submission**: From ~2 events/second to ~194 events/second
- **Concurrent processing**: Utilizes multiple CPU cores with worker pools
- **Efficient HTTP client**: Reuses connections and handles timeouts properly
- **Memory efficient**: Generates and processes events in batches
- **Better error handling**: Comprehensive error reporting and recovery
- **Persistent logging**: All progress and results saved to log files

## Installation

Build the tool from source:

```bash
go build -buildvcs=false -o bin/test-events ./cmd/test-events
```

## Usage

### Basic Usage

```bash
# Test with default settings (100,000 events)
./bin/test-events

# Test with custom parameters
./bin/test-events -events 50000 -workers 16 -url http://localhost:8080

# Test with verbose output
./bin/test-events -verbose -events 10000
```

### Command Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-url` | `http://localhost:9080` | Base URL of the service |
| `-events` | `100000` | Number of events to generate and submit |
| `-top` | `50` | Number of top entries to fetch from leaderboard |
| `-workers` | `CPU cores * 2` | Number of concurrent workers |
| `-timeout` | `30s` | HTTP request timeout |
| `-output` | `generated_events_TIMESTAMP.json` | Output file for generated events |
| `-log` | `test_log_TIMESTAMP.log` | Log file for test output |
| `-verbose` | `false` | Enable verbose logging |
| `-help` | - | Show help message |

### Examples

```bash
# Quick test with 1000 events
./bin/test-events -events 1000 -workers 4

# Stress test with maximum concurrency
./bin/test-events -events 100000 -workers 32 -timeout 60s

# Test against remote service
./bin/test-events -url https://api.example.com -events 50000

# Save events to specific file
./bin/test-events -events 10000 -output my_test_events.json

# Test with custom log file
./bin/test-events -events 50000 -log my_test.log

# Test with both custom output and log files
./bin/test-events -events 10000 -output events.json -log test.log
```

## Logging

The tool provides comprehensive logging capabilities:

### Automatic Logging
- **Default behavior**: All output is automatically logged to a timestamped file
- **Log format**: `test_log_YYYYMMDD_HHMMSS.log`
- **Dual output**: Logs appear both on console and in the log file
- **Timestamped entries**: Each log entry includes precise timestamps

### Custom Log Files
- **Custom filename**: Use `-log filename.log` to specify a custom log file
- **Persistent storage**: Log files are preserved for analysis and debugging
- **Complete record**: All progress updates, statistics, and results are logged

### Log Content
The log files contain:
- Configuration details
- Real-time progress updates (e.g., "Submitted: 183046/500000 (success: 72915, duplicate: 0, failed: 110131)")
- Performance statistics
- Error messages and warnings
- Final results and verification

### Example Log Output
```
2025/09/04 16:32:12.500621 📝 Logging to file: test_log_20250904_163212.log
2025/09/04 16:32:12.500975 🚀 Starting Cuju Event Test
📊 Configuration:
   Base URL: http://localhost:9080
   Events: 100000
   Workers: 16
   Timeout: 30s
   Top N: 50
   Log File: test_log_20250904_163212.log
   Verbose: false

2025/09/04 16:32:12.500996 🔍 Checking service health...
2025/09/04 16:32:12.504911 ✅ Service is healthy
2025/09/04 16:32:12.504912 🎲 Generating 100000 events with unique talent IDs...
2025/09/04 16:32:12.504913 ✅ Generated 100000 events successfully
2025/09/04 16:32:12.504914 📤 Submitting 100000 events with 16 workers...
📤 Submitted: 183046/100000 (success: 72915, duplicate: 0, failed: 110131)
📤 Submitted: 250000/100000 (success: 150000, duplicate: 0, failed: 100000)
...
```

## Output

The tool provides comprehensive output including:

1. **Configuration Summary**: Shows all test parameters
2. **Progress Reports**: Real-time progress during execution
3. **Performance Statistics**: Success rates, timing, and throughput
4. **Result Verification**: Top performers and consistency checks
5. **Event File**: Generated events saved to JSON file
6. **Log File**: Complete test execution log

### Sample Output

```
🚀 Starting Cuju Event Test
📊 Configuration:
   Base URL: http://localhost:9080
   Events: 1000
   Workers: 8
   Timeout: 30s
   Top N: 50
   Log File: test_log_20250904_163212.log
   Verbose: false

🔍 Checking service health...
✅ Service is healthy
🎲 Generating 1000 events with unique talent IDs...
✅ Generated 1000 events successfully
📤 Submitting 1000 events with 8 workers...
✅ Event submission completed:
   Successful: 1000
   Duplicate: 0
   Failed: 0
⏳ Waiting for events to be processed...
🏆 Retrieving rankings for 1000 talents with 8 workers...
✅ Ranking retrieval completed:
   Retrieved: 1000
   Failed: 0
🥇 Getting top 50 leaderboard entries...
✅ Retrieved 50 leaderboard entries
🔍 Verifying results...
✅ Leaderboard consistency verified
🏆 Top 10 performers from rankings:
   1. bd684547-c659-403e-b745-4a83b26e78a1 - Score: 8.488
   2. f9564f63-60c8-4aee-aa5a-f13b2f7fbff2 - Score: 8.487
   ...

📈 === FINAL STATISTICS ===
   Events Generated: 1000
   Events Submitted: 1000
   Events Successful: 1000
   Events Duplicate: 0
   Events Failed: 0
   Rankings Retrieved: 1000
   Leaderboard Entries: 50
   Total Duration: 5.145318958s
   Success Rate: 100.00%
   Events/Second: 194.35

✅ Test completed successfully!
```

## Architecture

The tool is built with a modular architecture:

- **`main.go`**: CLI interface and configuration
- **`config.go`**: Configuration structures and types
- **`runner.go`**: Main test orchestration
- **`generator.go`**: Concurrent event generation
- **`http.go`**: HTTP client and concurrent submission
- **`ranking.go`**: Concurrent ranking retrieval
- **`verification.go`**: Result verification and reporting

## Performance Tuning

For optimal performance:

1. **Worker Count**: Set to 2-4x CPU cores for I/O bound operations
2. **Timeout**: Adjust based on network latency and service response time
3. **Batch Size**: The tool automatically optimizes batch sizes based on worker count
4. **Memory**: Large event counts may require more memory for event storage
5. **Logging**: Use custom log files for long-running tests to avoid large default log files

## Error Handling

The tool includes comprehensive error handling:

- **Service Health Checks**: Verifies service availability before starting
- **HTTP Timeouts**: Configurable timeouts prevent hanging requests
- **Retry Logic**: Automatic retry for transient failures
- **Progress Reporting**: Shows failed operations in real-time
- **Graceful Degradation**: Continues operation even with some failures
- **Logging**: All errors are logged to files for debugging

## Comparison with Shell Script

| Aspect | Shell Script | Go CLI Tool |
|--------|-------------|-------------|
| **Performance** | ~2 events/sec | ~194 events/sec |
| **Concurrency** | Sequential | Multi-threaded |
| **Memory Usage** | High (shell + jq) | Optimized |
| **Error Handling** | Basic | Comprehensive |
| **Progress Reporting** | Limited | Real-time |
| **Configuration** | Hardcoded | Flexible |
| **Dependencies** | bash, curl, jq, uuidgen | Self-contained |
| **Logging** | Console only | Console + File |

## Requirements

- Go 1.24 or later
- Access to a running Cuju service
- Network connectivity to the service endpoint

## License

This tool is part of the Cuju project and follows the same license terms.