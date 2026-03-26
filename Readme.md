# compro-dam-tools

A lightweight HTTP API toolkit for DAM (Digital Asset Management) integrations by Communication Pro.

## Current Tools

### File Deleter
Accepts a list of product IDs and deletes the corresponding files from a configured directory in the background.

- `POST /delete` — deletes `{id}.jpg` and `preview{id}.jpg` for each provided ID

## Requirements

- Go 1.21 or newer (for building)
- Windows (deployment target)

## Configuration

All parameters are controlled via `config.yaml` in the same folder as the binary:

```yaml
port: 8080
files_dir: ./files
bearer_token: your-secret-token-here
max_ids: 20
max_body_size: 1024
delete_retries: 5
delete_retry_delay: 500ms
```

| Parameter | Description |
|---|---|
| `port` | Port the API listens on |
| `files_dir` | Directory where asset files are stored |
| `bearer_token` | Secret token for API authentication |
| `max_ids` | Maximum number of IDs allowed per request |
| `max_body_size` | Maximum request body size in bytes |
| `delete_retries` | How many times to retry a locked file |
| `delete_retry_delay` | Delay between retries (e.g. `500ms`, `1s`) |

## API Usage

### POST /delete

**Headers:**
```
Authorization: Bearer your-secret-token-here
Content-Type: application/json
```

**Body:**
```json
{
  "ids": ["123456", "987654321"]
}
```

**Response:**
```json
{"status": "ok"}
```

The API always responds `200 OK` immediately. File deletion happens in the background.

**Error responses:**

| Status | Reason |
|---|---|
| `400` | Invalid JSON, empty ID, invalid ID format, too many IDs |
| `401` | Missing or invalid bearer token |
| `405` | Wrong HTTP method |

## Building

### For Windows (from Linux)

```bash
GOOS=windows GOARCH=amd64 go build -o compro-dam-tools.exe
```

### For Windows (from Windows)

```bash
go build -o compro-dam-tools.exe
```

## Deployment

Copy these two files to the target machine:

```
compro-dam-tools.exe
config.yaml
```

No runtime or additional dependencies needed.

Run the binary:

```bash
compro-dam-tools.exe
```

## Development Setup

1. Install [Go 1.21+](https://go.dev/dl/)
2. Clone the repository:
   ```bash
   git clone https://github.com/communicationpro/compro-dam-tools.git
   cd compro-dam-tools
   ```
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. Run locally:
   ```bash
   go run main.go
   ```

## Project Structure

```
compro-dam-tools/
├── main.go        # Application source
├── config.yaml    # Configuration file
├── go.mod         # Go module definition
├── go.sum         # Dependency checksums
└── README.md      # This file
```

## License

Copyright © Communication Pro. All rights reserved.
