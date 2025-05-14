# GoQuest: A Concurrent, Programmable Task Orchestrator

## Overview
**GoQuest** is a standalone Go CLI tool and library that orchestrates concurrent task workflows defined in a simple Go-based script. It allows users to define tasks (e.g., HTTP requests, computations) and execute them in parallel with a configurable worker pool, leveraging Go's concurrency model (goroutines and channels). Designed as a lightweight alternative to tools like `make` or Apache Airflow, GoQuest is ideal for developers automating workflows, showcasing advanced Go features like concurrency, interfaces, and robust error handling.

This project is built by  me , **Abhinav Anand**, a third-year student at NIT Durgapur, to demonstrate proficiency in Go for backend and DevOps roles. The codebase emphasizes clean code, modularity, and testability, making it a standout portfolio piece for recruiters.

## Features
- **Concurrent Task Execution**: Runs multiple tasks in parallel using a worker pool with configurable concurrency.
- **Task Definition**: Supports tasks defined in a Go script with fields for name, execution logic, and dependencies (dependency management in progress).
- **CLI Interface**: Provides a user-friendly CLI with flags for script path (`--script`), concurrency (`--concurrency`), and verbose logging (`--verbose`).
- **Robust Error Handling**: Aggregates task errors and provides detailed, contextual error messages.
- **Thread-Safe Logging**: Outputs timestamped logs (INFO, DEBUG) with thread-safe operations for concurrent execution.
- **Extensible Design**: Uses interfaces (`TaskRunner`) for pluggable task types, preparing for future enhancements like HTTP or file tasks.
- **JSON Output**: Export task results as JSON for machine-readable results.

## Project Status
As of completion of Part 2 (of a 5-part development plan), GoQuest supports:
- Sequential and concurrent execution of multiple tasks defined in a Go script.
- A worker pool for parallel task execution with configurable concurrency.
- Thread-safe result collection and error aggregation.
- A CLI with flags for script loading, concurrency, and verbose debugging.
- Basic task validation and logging.

Future parts will add:
- Part 3: A DSL parser for dynamic script loading.
- Part 4: Task dependency management.
- Part 5: Advanced features like JSON output and built-in task types (e.g., HTTP requests).

## Architecture
GoQuest follows a modular, clean architecture with a clear separation of concerns:
- **CLI (`cmd/goquest`)**: Handles user input, parses flags, and orchestrates task execution.
- **Task Package (`pkg/task`)**: Defines task specifications (`TaskSpec`), interfaces (`TaskRunner`), and the concurrent executor (`Executor`).
- **Utility Package (`internal/util`)**: Provides thread-safe logging and error handling helpers.
- **Scripts (`scripts`)**: Contains user-defined Go scripts with task definitions.

The task execution pipeline:
1. The CLI loads a script (e.g., `scripts/example.go`) and registers tasks in a global registry.
2. The `Executor` creates a worker pool of goroutines, dispatching tasks via a channel.
3. Workers execute tasks concurrently, sending results to a result channel.
4. Results are collected thread-safely, and errors are aggregated for reporting.
5. The CLI displays task summaries and logs execution details.

## Folder Structure
```
goquest/
├── cmd/
│   └── goquest/
│       └── main.go           # CLI entry point
├── pkg/
│   └── task/
│       ├── task.go          # TaskSpec, TaskRunner, and registry logic
│       ├── executor.go      # Concurrent task executor and worker pool
│       └── task_test.go     # Tests for task logic
├── internal/
│   └── util/
│       ├── log.go           # Thread-safe logging functions
│       ├── error.go         # Error wrapping and exit helpers
│       ├── log_test.go      # Tests for logging
│       └── error_test.go    # Tests for error handling
├── scripts/
│   └── example.go           # Sample task script
├── .gitignore               # Git ignore file
├── go.mod                   # Go module file
├── go.sum                   # Dependency checksums
├── README.md                # Project documentation
```

## Prerequisites
- **Go**: Version 1.22 or higher (tested with 1.22).
- **Git**: For cloning the repository.
- **Optional Tools**:
  - `golangci-lint`: For linting (install with `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`).
  - `goimports`: For code formatting (install with `go install golang.org/x/tools/cmd/goimports@latest`).

## Setup Instructions
Follow these steps to set up and run GoQuest on your local machine.

### 1. Clone the Repository
```bash
git clone https://github.com/AbhinavAnand241201/goquest.git
cd goquest
```

### 2. Initialize Go Module
Ensure the module is initialized and dependencies are downloaded:
```bash
go mod tidy
```

### 3. Build the CLI
Compile the CLI binary:
```bash
go build -o bin/goquest cmd/goquest/main.go
```

### 4. Run Tests
Verify the codebase with unit tests:
```bash
go test ./pkg/task/... ./internal/util/...
```

### 5. Install Optional Tools (Recommended)
For linting and formatting:
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
```

## Usage
GoQuest runs tasks defined in a Go script via the CLI. The sample script `scripts/example.go` defines three tasks for testing.

### Basic Command
Run all tasks in `scripts/example.go` with default concurrency (4):
```bash
./bin/goquest run --script scripts/example.go
```

**Example Output**:
```
2025-05-14 13:35:00 INFO: Loading script: scripts/example.go
2025-05-14 13:35:00 INFO: Running 3 tasks with concurrency 4
2025-05-14 13:35:01 INFO: Task task1: success, result: Hello from task1, duration: 0.1s
2025-05-14 13:35:01 INFO: Task task2: success, result: Delayed task, duration: 1.0s
2025-05-14 13:35:01 INFO: Task task3: success, result: 42, duration: 0.1s
2025-05-14 13:35:01 INFO: 3 tasks completed, 0 failed
```

### Custom Concurrency
Run tasks with a specific concurrency level (e.g., 8):
```bash
./bin/goquest run --script scripts/example.go --concurrency 8
```

### Verbose Mode
Enable debug logging for detailed execution info:
```bash
./bin/goquest run --script scripts/example.go --verbose
```

**Example Verbose Output**:
```
2025-05-14 13:35:00 INFO: Loading script: scripts/example.go
2025-05-14 13:35:00 DEBUG: Dispatching task: task1
2025-05-14 13:35:00 DEBUG: Dispatching task: task2
2025-05-14 13:35:00 DEBUG: Dispatching task: task3
2025-05-14 13:35:01 DEBUG: Received result for task: task1
2025-05-14 13:35:01 INFO: Task task1: success, result: Hello from task1, duration: 0.1s
...
```

### Writing Custom Scripts
Create a new script (e.g., `scripts/myscript.go`) to define tasks. Example:
```go
package main

import (
    "context"
    "github.com/AbhinavAnand241201/goquest/pkg/task"
)

func init() {
    task.RegisterTasks([]task.TaskSpec{
        {
            Name: "custom_task",
            Run: func(ctx context.Context) (any, error) {
                return "Custom task executed", nil
            },
            Depends: []string{},
        },
    })
}
```

Run the custom script:
```bash
./bin/goquest run --script scripts/myscript.go
```

## Development Details
GoQuest is developed in five parts to ensure a structured, incremental approach. As of Part 2, the following are complete:

### Part 1: Project Setup, Task Definition, and Basic CLI
- Initialized Go module and Git repository.
- Defined `TaskSpec` (name, run function, dependencies) and `TaskRunner` interface.
- Created a basic CLI to run a single task sequentially.
- Set up thread-safe logging and error handling in `internal/util`.

### Part 2: Adding Concurrency with Goroutines and Channels
- Extended task registry to support multiple tasks with validation.
- Implemented a concurrent `Executor` with a worker pool using goroutines and channels.
- Added thread-safe result collection and error aggregation.
- Updated CLI to run multiple tasks with `--concurrency` and `--verbose` flags.
- Added tests for concurrent execution and logging.

### Upcoming Parts
- **Part 3**: Implement a DSL parser to dynamically load tasks from Go scripts.
- **Part 4**: Add task dependency management for ordered execution.
- **Part 5**: Introduce advanced features like JSON output, built-in task types (e.g., HTTP, file operations), and performance optimizations.

### Key Go Features Used
- **Goroutines and Channels**: For concurrent task execution and result collection.
- **Interfaces**: `TaskRunner` for extensible task types.
- **Context**: For task cancellation and timeouts.
- **Mutexes**: For thread-safe logging and result aggregation.
- **Flag Package**: For CLI argument parsing.
- **Testing**: Comprehensive unit tests for task execution and utilities.

## Testing
The project includes unit tests to ensure reliability:
- **Task Tests (`pkg/task/task_test.go`)**: Verify task validation, registry operations, and executor behavior.
- **Executor Tests (`pkg/task/executor_test.go`)**: Test concurrent task execution with varying task counts and durations.
- **Utility Tests (`internal/util/log_test.go`, `error_test.go`)**: Validate thread-safe logging and error handling.

Run all tests:
```bash
go test ./...
```

## Linting and Formatting
Ensure code quality with linting and formatting:
```bash
golangci-lint run
goimports -w .
```

## Troubleshooting
- **Issue**: `go build` fails with missing dependencies.
  - **Fix**: Run `go mod tidy` to download dependencies.
- **Issue**: CLI output is interleaved or unreadable.
  - **Fix**: Enable `--verbose` to debug or check `internal/util/log.go` for mutex issues.
- **Issue**: Tasks hang or don't complete.
  - **Fix**: Verify the `--concurrency` value is reasonable (e.g., 4–8) and check script tasks respect context timeouts.
- **Issue**: Tests fail due to race conditions.
  - **Fix**: Run `go test -race` to detect and fix race conditions in `Executor` or logging.

## Contributing
Contributions are welcome! To contribute:
1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/my-feature`).
3. Commit changes (`git commit -m "Add my feature"`).
4. Push to the branch (`git push origin feature/my-feature`).
5. Open a pull request with a detailed description.

Please follow Go best practices, run tests, and lint the code before submitting.

## Future Enhancements
- **Dynamic DSL**: Parse Go scripts dynamically without static registration (Part 3).
- **Dependency Graphs**: Execute tasks in the correct order based on dependencies (Part 4).
- **Built-in Tasks**: Support common tasks like HTTP requests or file operations (Part 5).
- **JSON Output**: Export task results as JSON for integration with other tools.
- **Benchmarks**: Add performance benchmarks to showcase scalability.

## Contact
For questions or feedback, contact Abhinav Anand via:
- **GitHub**: [AbhinavAnand241201](https://github.com/AbhinavAnand241201)

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details. 
