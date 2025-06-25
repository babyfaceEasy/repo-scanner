# repo-scanner

## Introduction
`repo-scanner` is a command-line tool written in Go that scans GitHub repositories to identify and report files exceeding a specified size threshold. The application fetches repositories via the GitHub API, extracts files, scans them locally, and outputs results in JSON format.

Key features:
- **Structured Logging**: Uses `zerolog` for JSON or human-readable logs, configurable via environment variables, implemented in a reusable `pkg/logger` package.
- **Fault Tolerance**: Handles GitHub API rate limits with a retry decorator and exponential backoff.
- **Modular Design**: Employs dependency injection and interfaces for testability and extensibility.
- **Environment Management**: Centralizes configuration in an `env` package for robust `.env` handling.
- **Docker Support**: Packaged as a lightweight Docker image for consistent deployment.

This project showcases production-grade Go practices, including clean architecture, comprehensive tests, and observability, making it suitable for real-world use cases.

## Table of Contents
- [repo-scanner](#repo-scanner)
  - [Introduction](#introduction)
  - [Table of Contents](#table-of-contents)
  - [Prerequisites](#prerequisites)
  - [Setup Instructions](#setup-instructions)
  - [Usage Examples](#usage-examples)
    - [Running Locally](#running-locally)
    - [Running with Docker](#running-with-docker)
    - [Development Mode (Human-Readable Logs)](#development-mode-human-readable-logs)
    - [Handling Rate Limits](#handling-rate-limits)
  - [Architecture](#architecture)
  - [Data Flow](#data-flow)
  - [Development](#development)
    - [Folder Structure](#folder-structure)
    - [Dependencies](#dependencies)
  - [Testing](#testing)

## Prerequisites
- **Go**: Version 1.20 or higher (for local development and testing).
- **Docker**: For building and running the application in a container.
- **GitHub Personal Access Token**: Required for GitHub API access (with `repo` scope for private repositories).
- **Git**: For cloning the repository.
- A `.env` file with configuration (see [Setup Instructions](#setup-instructions)).

## Setup Instructions

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/yourusername/repo-scanner.git
   cd repo-scanner
   ```

2. **Set Up Environment Variables**:
   - Create a `.env` file in the project root:
     ```plaintext
     GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
     LOG_ENV=production
     ```
   - Replace `ghp_xxx` with your GitHub personal access token (generate one at [GitHub Settings > Developer Settings > Personal Access Tokens](https://github.com/settings/tokens)).
   - `LOG_ENV` can be `production` (JSON logs) or `development` (human-readable logs).

3. **Install Dependencies** (for local development):
   ```bash
   go mod tidy
   ```

4. **Build the Application** (optional, for local runs):
   ```bash
   go build -o repo-scanner ./cmd/repo-scanner
   ```

5. **Build Docker Image** (for containerized runs):
   ```bash
   docker build -t repo-scanner .
   ```

## Usage Examples

### Running Locally
Scan a repository for files larger than 1MB:
```bash
./repo-scanner scan '{"clone_url":"https://github.com/owner/repo.git","size":1.0}'
```
**Output** (JSON):
```json
{
  "total": 2,
  "files": [
    {"name": "large_file.zip", "size": 1500000},
    {"name": "sub/dir/big_image.png", "size": 1200000}
  ]
}
```

### Running with Docker
```bash
docker run --env-file .env repo-scanner scan '{"clone_url":"https://github.com/owner/repo.git","size":1.0}'
```

### Development Mode (Human-Readable Logs)
Set `LOG_ENV=development` in `.env` and run:
```bash
./repo-scanner scan '{"clone_url":"https://github.com/owner/repo.git","size":1.0}'
```
**Log Output** (example):
```
2025-06-25 11:25:05 INFO  Config parsed clone_url=https://github.com/owner/repo.git size_mb=1
2025-06-25 11:25:06 INFO  Repository downloaded path=/tmp/repo-scan-12345
2025-06-25 11:25:07 INFO  Found large file name=large_file.zip size=1500000
```

### Handling Rate Limits
If the GitHub API returns a 429 (rate limit), the application retries up to 3 times with exponential backoff (500ms base delay + jitter). Logs will show retry attempts:
```json
{"level":"info","attempt":1,"delay_ms":510,"error":"rate limit exceeded","message":"Retrying after delay"}
```

## Architecture

The application follows a modular, clean architecture with dependency injection, ensuring testability and extensibility. Below is a high-level diagram of the component interactions:

```mermaid
classDiagram
    class Main {
        +Execute CLI
    }
    class Env {
        +Load() Config
    }
    class Logger {
        +Info()
        +Error()
        +Warn()
        +Debug()
        +Fatal()
    }
    class Service {
        +Scan()
    }
    class GitHubClient {
        <<interface>>
        +DownloadRepo()
    }
    class GitHub {
        +DownloadRepo()
    }
    class Retrier {
        +DownloadRepo()
    }
    class Scanner {
        +Scan()
    }
    class Config {
        +Parse()
    }
    class Output {
        +Write()
    }

    Main --> Env : Loads
    Main --> Logger : Initializes
    Main --> Service : Creates
    Main --> GitHub : Creates
    Main --> Retrier : Wraps GitHub
    Service --> Config : Parses input
    Service --> GitHubClient : Downloads repo
    Service --> Scanner : Scans files
    Service --> Output : Writes results
    Service --> Logger : Logs
    Retrier --> GitHubClient : Decorates
    GitHub --> Logger : Logs
    Scanner --> Logger : Logs
    Retrier --> Logger : Logs

    note for GitHubClient "Interface for GitHub interactions"
    note for Retrier "Retry decorator with backoff"
    note for Logger "Reusable logging package in pkg"
```

**Key Components**:
- **Main**: Entry point, initializes dependencies and runs the Cobra CLI.
- **Env**: Loads and validates environment variables (`GITHUB_TOKEN`, `LOG_ENV`).
- **Logger**: `zerolog`-based logging with JSON (production) or console (development) output, designed for reuse in `pkg/logger`.
- **Service**: Orchestrates business logic, coordinating config parsing, repo download, scanning, and output.
- **GitHubClient**: Interface for GitHub API interactions, implemented by `GitHub`.
- **Retrier**: Decorator that adds retry logic with exponential backoff for `GitHubClient`.
- **Scanner**: Traverses extracted repository files to identify large files.
- **Config**: Parses JSON input (`clone_url`, `size`).
- **Output**: Writes scan results as JSON to stdout.

**Note**: The `logger` package is placed in `pkg` to emphasize its potential reusability across projects, providing a standardized logging interface backed by `zerolog`.

## Data Flow

The following sequence diagram illustrates how data flows through the application when the `scan` command is executed, from CLI input to JSON output:

```mermaid
sequenceDiagram
    actor User
    User->>CLI: scan {"clone_url":"...", "size":1.0}
    CLI->>Env: Load()
    Env-->>CLI: Config
    CLI->>Logger: New()
    Logger-->>CLI: Logger
    CLI->>Service: New()
    Service->>Config: Parse(JSON)
    Config-->>Service: Config struct
    Service->>Logger: Log("Config parsed")
    Service->>GitHubClient: DownloadRepo()
    GitHubClient->>Retrier: DownloadRepo()
    Retrier->>GitHub: DownloadRepo()
    GitHub->>Logger: Log("Fetching tarball")
    GitHub-->>Retrier: Tarball
    Retrier-->>Service: Temp dir
    Service->>Logger: Log("Repository downloaded")
    Service->>Scanner: Scan(dir, threshold)
    Scanner->>Logger: Log("Found large file")
    Scanner-->>Service: File list
    Service->>Logger: Log("File scan completed")
    Service->>Output: Write()
    Output-->>User: JSON output
```

**Data Flow Description**:
1. The user invokes the `scan` command with JSON input (e.g., `{"clone_url":"https://github.com/owner/repo.git","size":1.0}`).
2. The CLI loads environment variables via `Env` and initializes the `Logger`.
3. The `Service` parses the JSON input using `Config` and logs the action.
4. The `Service` requests the repository download via `GitHubClient`, which is decorated by `Retrier` for rate limit handling.
5. The `GitHub` package fetches and extracts the repository, logging progress, and returns a temporary directory.
6. The `Service` scans the directory using `Scanner`, which logs and collects large files.
7. The `Service` writes the results as JSON via `Output`, which is displayed to the user.

## Development

### Folder Structure
```
repo-scanner/
├── cmd/
│   └── repo-scanner/
│       └── main.go              # CLI entry point
├── internal/
│   ├── config/                 # JSON input parsing
│   ├── env/                    # Environment variable management
│   ├── github/                 # GitHub API client
│   ├── model/                  # Data structures
│   ├── output/                 # JSON output
│   ├── retry/                  # Retry decorator
│   ├── scanner/                # File scanning
│   ├── service/                # Business logic
├── pkg/
│   ├── logger/                 # Reusable logging package
├── .env                        # Environment variables
├── go.mod                      # Go module
├── Dockerfile                  # Docker configuration
└── README.md                   # Documentation
```

### Dependencies
- `github.com/joho/godotenv`: Loads `.env` files.
- `github.com/rs/zerolog`: Structured logging.
- `github.com/spf13/cobra`: CLI framework.

Install dependencies:
```bash
go mod tidy
```

## Testing

The project includes comprehensive unit tests for all packages, achieving high coverage. Run tests:
```bash
go test ./internal/... ./pkg/...
```

Key test areas:
- `env`: Validates `.env` loading and error cases.
- `logger`: Verifies log levels and output formats.
- `github`: Mocks GitHub API responses for `DownloadRepo`.
- `retry`: Tests retry logic with success, partial failure, and max retries.
- `scanner`: Simulates file systems to verify large file detection.
- `service`: Integrates components with mocked dependencies.
- `config`, `model`, `output`: Validate parsing, serialization, and output.
