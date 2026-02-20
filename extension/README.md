# LogViewer VS Code Extension

A VS Code extension that provides a graphical interface for viewing and searching logs from multiple backends (OpenSearch, Splunk, CloudWatch, Kubernetes, Docker, SSH).

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    VS Code Extension                         │
│  ┌─────────────────┐    ┌─────────────────────────────────┐ │
│  │  ServerManager  │───▶│  LogViewer Go Binary            │ │
│  │  (spawns bin)   │    │  (HTTP API on localhost:PORT)   │ │
│  └─────────────────┘    └─────────────────────────────────┘ │
│           │                            ▲                     │
│           ▼                            │                     │
│  ┌─────────────────────────────────────┴───────────────────┐│
│  │              Webview Panel (Svelte SPA)                 ││
│  │  - Fetches from http://localhost:PORT                   ││
│  │  - Port injected via window.LOGVIEWER_PORT              ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Supported Backends

- OpenSearch / Elasticsearch
- Splunk
- AWS CloudWatch
- Kubernetes (via kubectl)
- Docker
- SSH (remote log files)

## Features

### Sidebar Search Panel
- **Context Selector**: Dropdown to select log context (environment, service, etc.)
- **Quick Filters**: Pill buttons for ERROR, WARN, INFO levels
- **Time Range Selector**: 15m, 1h, 24h, 7d options
- **Size Selector**: Number of results to fetch (50, 100, 250, 500, 1000)
- **Query Input**: Native query string support (e.g., Splunk SPL, OpenSearch DSL)
- **Recent Searches**: Saved in localStorage for quick access

### Results Panel
- **Edge-to-edge Table**: Compact design using full width
- **Color-coded Levels**: Visual distinction for ERROR (red), WARN (yellow), INFO (blue), etc.
- **Infinite Scroll Pagination**: Automatic loading when scrolling down
- **Load More Button**: Clickable button to manually trigger loading next page
- **Row Expansion**: Click to expand and view full JSON details
- **Toolbar Actions**:
  - 🔄 Refresh: Re-run the current query
  - ⬇️ Export: Download logs as JSON file
  - 📋 Copy: Copy logs to clipboard
- **Footer Status**: Shows result count and pagination status

### Tab Management
- **Hash-based Reuse**: Tabs with identical search parameters are reused
- **Multiple Tabs**: Different searches open in separate tabs
- **Auto-reveal**: Clicking search with existing tab reveals it instead of creating duplicate

---

## Usage

### Installation

Install the extension from a `.vsix` file:

```bash
code --install-extension logviewer-0.0.1.vsix
```

### Configuration

1. Create a LogViewer configuration file (`config.yaml`) - see main LogViewer documentation
2. Set the path in VS Code settings:

```json
{
  "logviewer.configPath": "/path/to/config.yaml"
}
```

### Commands

| Command | Description |
|---------|-------------|
| `LogViewer: Open` | Open the LogViewer panel |
| `LogViewer: Restart Server` | Restart the backend server |
| `LogViewer: Show Server Output` | View server logs |

### Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `logviewer.configPath` | `""` | Path to LogViewer configuration file |
| `logviewer.autoStart` | `true` | Automatically start server when opening panel |

---

## Development

All development tasks are run via Makefile from the **project root** directory.

### Prerequisites

- Node.js 18+
- Go 1.21+
- VS Code 1.85+

### Quick Start (Development)

```bash
# From project root - builds everything for development
make extension/dev/setup
```

This will:
1. Build the Go binary for your current platform
2. Build the web frontend
3. Copy assets to the extension
4. Compile the extension TypeScript

Then open the `extension/` folder in VS Code and press **F5** to launch the Extension Development Host.

### Available Makefile Targets

Run these from the **project root**:

```bash
# Install dependencies only
make extension/install

# Compile TypeScript only
make extension/compile

# Watch mode (recompile on changes)
make extension/watch

# Full build (Go binaries + web + extension)
make extension/build

# Package as .vsix
make extension/package

# Clean build artifacts
make extension/clean
```

### Project Structure

```
extension/
├── src/
│   ├── extension.ts       # Entry point, command registration
│   ├── ServerManager.ts   # Go binary lifecycle management
│   └── LogViewerPanel.ts  # Webview provider, CSP, port injection
├── bin/                   # Go binaries (populated by make)
├── media/                 # Web frontend dist (populated by make)
├── out/                   # Compiled JavaScript
├── package.json           # Extension manifest
└── tsconfig.json          # TypeScript configuration
```

### Key Implementation Details

1. **Binary Selection**: `ServerManager.ts` selects the correct binary based on `process.platform` and `process.arch`

2. **Port Discovery**: Server spawns with `--port 0` (random port), extension parses stdout for actual port

3. **Port Injection**: `LogViewerPanel.ts` injects `window.LOGVIEWER_PORT` into the webview HTML

4. **CSP**: Content Security Policy allows `connect-src http://localhost:${port}` for API calls

5. **Pagination**: Backend supports pagination via `pageToken` (request) and `nextPageToken` (response). Frontend automatically passes token when scrolling or clicking "Load more" button.

6. **VS Code API**: Webview can only acquire VS Code API once. It's stored at module level and reused for all message passing.

7. **Svelte 5 Runes**: Uses modern `$state`, `$effect`, and `$props` instead of legacy stores and lifecycle methods.

### Debugging

1. Open `extension/` folder in VS Code
2. Press F5 to launch Extension Development Host
3. Open Command Palette → "LogViewer: Open"
4. Use "LogViewer: Show Server Output" to view server logs

### Troubleshooting

**"Frontend not built" error**
```bash
make web/build
make extension/media
```

**"Binary not found" error**
```bash
make build  # or make extension/binaries for all platforms
```

**Server not starting**
- Check the LogViewer Server output channel
- Verify `logviewer.configPath` is set correctly
- Ensure the config file exists and is valid

---

## Building for Distribution

```bash
# Build everything and package as .vsix
make vscode

# Output: extension/logviewer-0.0.1.vsix
```

The `.vsix` file includes:
- Pre-compiled Go binaries for all platforms (darwin/linux/windows, amd64/arm64)
- Pre-built web frontend
- Compiled extension JavaScript

## License

MIT
