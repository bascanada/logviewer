# LogViewer Web Frontend

A Svelte-based single-page application (SPA) that provides the user interface for the LogViewer VS Code extension. It communicates with the LogViewer Go backend via HTTP API.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Svelte Frontend                          │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  App.svelte                                             ││
│  │  ├── ContextSelector (GET /contexts)                    ││
│  │  ├── SearchBar (triggers queries)                       ││
│  │  └── LogTable (POST /query/logs)                        ││
│  │      └── LogRow (expandable details)                    ││
│  └─────────────────────────────────────────────────────────┘│
│                           │                                  │
│                           ▼                                  │
│              http://localhost:${LOGVIEWER_PORT}              │
│                           │                                  │
└───────────────────────────┼──────────────────────────────────┘
                            ▼
                  LogViewer Go Backend
```

## Features

- **Context Selector**: Switch between log sources (OpenSearch, Splunk, etc.)
- **Search Bar**: Query input with time range and level filters
- **Log Table**: Displays results with infinite scroll pagination
- **Log Row**: Expandable rows showing full JSON details

---

## Development

All development tasks are run via Makefile from the **project root** directory.

### Prerequisites

- Node.js 18+

### Quick Start

```bash
# From project root - start development server
make web/dev
```

This starts the Vite dev server at `http://localhost:5173`.

**Note**: For the frontend to work, you need the LogViewer backend running:

```bash
# In another terminal - start the backend
make build
./build/logviewer server --port 8080 --config path/to/config.yaml
```

### Available Makefile Targets

Run these from the **project root**:

```bash
# Install dependencies
make web/install

# Start development server (hot reload)
make web/dev

# Build for production
make web/build

# Run type check
make web/check

# Run linter
make web/lint

# Clean build artifacts
make web/clean
```

### Project Structure

```
web/
├── src/
│   ├── lib/
│   │   ├── api/
│   │   │   ├── client.ts    # API client singleton
│   │   │   ├── types.ts     # TypeScript interfaces
│   │   │   └── index.ts     # Re-exports
│   │   ├── stores/
│   │   │   ├── context.ts   # Selected context store
│   │   │   ├── logs.ts      # Log entries store
│   │   │   └── search.ts    # Search state store
│   │   └── components/
│   │       ├── ContextSelector.svelte
│   │       ├── SearchBar.svelte
│   │       ├── LogTable.svelte
│   │       └── LogRow.svelte
│   ├── App.svelte           # Main application
│   └── main.ts              # Entry point
├── vite.config.ts           # Vite configuration
├── tsconfig.json            # TypeScript configuration
└── package.json
```

---

## API Contract

The frontend consumes these endpoints from the LogViewer Go backend:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/contexts` | GET | List available log contexts |
| `/query/logs` | POST | Query log entries |
| `/query/fields` | POST | Discover available fields |
| `/health` | GET | Health check |

### Port Configuration

The frontend reads the backend port from `window.LOGVIEWER_PORT`:

```typescript
// In src/lib/api/client.ts
const port = window.LOGVIEWER_PORT || 8080;
const BASE_URL = `http://localhost:${port}`;
```

- **In VS Code**: The extension injects this value via script tag
- **In development**: Defaults to `8080`

---

## Build Configuration

The build is configured for VS Code webview embedding:

```typescript
// vite.config.ts
export default defineConfig({
  base: './',  // Relative paths for webview
  resolve: {
    alias: {
      $lib: path.resolve('./src/lib'),
    },
  },
  build: {
    outDir: 'dist',
    rollupOptions: {
      output: {
        entryFileNames: 'assets/[name].js',
        chunkFileNames: 'assets/[name].js',
        assetFileNames: 'assets/[name].[ext]',
      },
    },
  },
});
```

---

## Customization

### Styling

Components use CSS custom properties for VS Code theme integration:

```css
/* VS Code theme variables (with fallbacks for standalone use) */
background: var(--vscode-editor-background, #1e1e1e);
color: var(--vscode-editor-foreground, #d4d4d4);
border-color: var(--vscode-panel-border, #333333);
```

### Adding New Components

1. Create component in `src/lib/components/`
2. Use existing stores or create new ones in `src/lib/stores/`
3. Import and use in `App.svelte`

### Adding New API Endpoints

1. Add types to `src/lib/api/types.ts`
2. Add method to `src/lib/api/client.ts`
3. Call from component

---

## Building for Extension

The web frontend is built and copied to the extension automatically:

```bash
# From project root
make extension/media
```

This runs `npm run build` and copies `dist/*` to `extension/media/`.

## License

MIT
