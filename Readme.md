# compro-dam-tools

A lightweight HTTP API toolkit for DAM (Digital Asset Management) integrations by Communication Pro.

## Current Tools

### File Deleter
Accepts a list of product IDs and deletes the corresponding files from a configured directory in the background.

- `POST /delete` — deletes `{id}.jpg` and `preview{id}.jpg` for each provided ID

### File Refresh
Deletes all files in the configured directory that are older than 2 days. For files matching `{id}.jpg` or `preview{id}.jpg`, it also calls the DAM API to request a fresh version.

- `POST /refresh` — scans the directory and refreshes old assets

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
dam_base_url: https://images.dam.com/images
dam_bearer_token: your-dam-token-here
dam_timeout: 2s
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
| `dam_base_url` | Base URL of the DAM API (e.g. `https://images.dam.com/images`) |
| `dam_bearer_token` | Bearer token for the DAM API |
| `dam_timeout` | Max time to wait for DAM API response (e.g. `2s`) |

## API Usage

All endpoints require a bearer token in the `Authorization` header:
```
Authorization: Bearer your-secret-token-here
```

### POST /delete

Deletes `{id}.jpg` and `preview{id}.jpg` for each provided ID. Responds immediately, deletion happens in the background.

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

**Error responses:**

| Status | Reason |
|---|---|
| `400` | Invalid JSON, empty ID, invalid ID format, too many IDs |
| `401` | Missing or invalid bearer token |
| `405` | Wrong HTTP method |

---

### POST /refresh

Scans `files_dir` and deletes all files older than 2 days. For files matching `{id}.jpg` or `preview{id}.jpg`, calls the DAM API to request a fresh version. Responds immediately, processing happens in the background.

**Headers:**
```
Authorization: Bearer your-secret-token-here
```

**Response:**
```json
{"status": "ok"}
```

**Refresh behavior:**
- Any file older than 2 days is deleted (regardless of name)
- Only files matching `{id}.jpg` or `preview{id}.jpg` trigger a DAM API call
- DAM API calls time out after `dam_timeout` and are logged if they fail
- Files that cannot be deleted are retried up to `delete_retries` times

**Error responses:**

| Status | Reason |
|---|---|
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
├── main.go             # Application source
├── config.yaml         # Configuration file (not in version control)
├── config.yaml.example # Configuration template
├── go.mod              # Go module definition
├── go.sum              # Dependency checksums
└── README.md           # This file
```

## License

Copyright © Communication Pro. All rights reserved.

## Running as a Windows Service

The recommended way to run compro-dam-tools as a Windows service is using [NSSM (Non-Sucking Service Manager)](https://nssm.cc/download).

### Setup

**1. Download NSSM**

Download the 64-bit version from https://nssm.cc/download and place `nssm.exe` somewhere permanent like `C:\tools\nssm\`.

**2. Create the deployment folder**

```powershell
mkdir C:\compro-dam-tools\logs
```

Copy `compro-dam-tools.exe` and `config.yaml` to `C:\compro-dam-tools\`.

**3. Install the service** (run as Administrator)

```powershell
nssm install compro-dam-tools "C:\compro-dam-tools\compro-dam-tools.exe"
nssm set compro-dam-tools AppDirectory "C:\compro-dam-tools"
nssm set compro-dam-tools DisplayName "Communication Pro DAM Tools"
nssm set compro-dam-tools Description "DAM integration toolkit by Communication Pro"
nssm set compro-dam-tools Start SERVICE_AUTO_START
```

**4. Configure logging**

```powershell
nssm set compro-dam-tools AppStdout "C:\compro-dam-tools\logs\service.log"
nssm set compro-dam-tools AppStderr "C:\compro-dam-tools\logs\error.log"
nssm set compro-dam-tools AppRotateFiles 1
nssm set compro-dam-tools AppRotateBytes 10485760
```

Logs rotate automatically at 10 MB.

**5. Start the service**

```powershell
nssm start compro-dam-tools
```

### Managing the Service

```powershell
nssm start compro-dam-tools      # start
nssm stop compro-dam-tools       # stop
nssm restart compro-dam-tools    # restart
nssm status compro-dam-tools     # check status
nssm remove compro-dam-tools     # uninstall
```

### Deployment Folder Structure

```
C:\compro-dam-tools\
├── compro-dam-tools.exe
├── config.yaml
└── logs\
    ├── service.log
    └── error.log
```

### Updating the Binary

```powershell
nssm stop compro-dam-tools
# replace compro-dam-tools.exe with the new version
nssm start compro-dam-tools
```
