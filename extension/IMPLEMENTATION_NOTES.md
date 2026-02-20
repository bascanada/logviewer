# Implementation Notes

This document details the implementation decisions and bug fixes during the development of the LogViewer VS Code extension.

## Development Timeline

### Phase 1: Initial Setup
- Created Svelte web UI with sidebar and results panels
- Implemented VS Code extension structure
- Set up build pipeline (web → extension/media, Go → extension/bin)

### Phase 2: UI Implementation
- Implemented compact VS Code-native styling
- Added quick filter pills (ERROR/WARN/INFO)
- Added context info chips
- Implemented edge-to-edge table design with color-coded levels
- Added row expansion for JSON details
- Added toolbar buttons (refresh, export, copy)

### Phase 3: Bug Fixes

#### Bug 1: VS Code API Acquisition Error
**Issue**: "An instance of the VS Code API has already been acquired"

**Root Cause**: Called `acquireVsCodeApi()` inside `handleSearch()` on every button click.

**Fix**: Moved API acquisition to module level, stored in variable, reused for all message posts.

```typescript
// web/src/lib/components/Sidebar.svelte
let vscode: any = null;
if (typeof acquireVsCodeApi !== 'undefined') {
  vscode = acquireVsCodeApi();
}
```

#### Bug 2: Search Button Only Worked Once
**Issue**: After first search, subsequent searches did nothing.

**Root Cause**: VS Code was reusing webview panels improperly.

**Fix**: Implemented hash-based tab management system that checks if tab exists before creating new one.

```typescript
// extension/src/LogViewerResults.ts
private static createHash(searchParams: SearchParams): string {
  const parts = [
    searchParams.contextId,
    searchParams.query || '',
    searchParams.timeRange || '1h',
    searchParams.level || '',
    searchParams.size?.toString() || '100'
  ];
  return parts.join('|');
}
```

#### Bug 3: Infinite Scroll Not Triggering
**Issue**: User scrolled to bottom but no more logs loaded.

**Root Cause**: IntersectionObserver wasn't set up properly with Svelte 5's reactivity.

**Fix**: Changed from `onMount` to `$effect` for reactive setup.

```typescript
// web/src/lib/components/Results.svelte
$effect(() => {
  if (sentinel && !observer) {
    observer = new IntersectionObserver(...);
    observer.observe(sentinel);
  }
  return () => observer?.disconnect();
});
```

#### Bug 4: No Pagination Token from Backend
**Issue**: Console showed `nextPageToken: undefined`, preventing pagination.

**Root Cause**: Backend was discarding pagination token with `entries, _, err := searchResult.GetEntries()`.

**Fix**: Used `searchResult.GetPaginationInfo()` method to get pagination info.

```go
// pkg/server/handlers.go
var nextPageToken *string
if paginationInfo := searchResult.GetPaginationInfo(); paginationInfo != nil && paginationInfo.NextPageToken != "" {
    nextPageToken = &paginationInfo.NextPageToken
}
```

Also needed to pass `PageToken` from request to LogSearch:

```go
if req.PageToken != "" {
    req.Search.PageToken.S(req.PageToken)
}
```

### Phase 4: UX Improvements

#### Reversed Log Order
Changed log order so newest logs appear first (at top), with pagination loading older logs when scrolling down.

```typescript
// Reverse logs so newest appear first
const reversedLogs = [...response.logs].reverse();
```

#### Clickable Load More Button
Changed passive "Scroll down for more" text to active "Load more logs ↓" button that scrolls to trigger loading.

```typescript
function handleLoadMore() {
  if (sentinel) {
    sentinel.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  }
}
```

## Technical Decisions

### Why Svelte 5?
- Modern reactive syntax with runes ($state, $effect, $props)
- Smaller bundle size compared to React/Vue
- Simpler mental model for small UI components
- Good TypeScript support

### Why Separate sidebar.html and results.html?
- Different entry points allow code splitting
- Sidebar and results have different lifecycles
- Reduces initial bundle size for each panel

### Why Hash-based Tab Management?
- Avoids duplicate tabs for same search
- Provides natural grouping by search parameters
- Easy to implement with deterministic hash function
- Users can re-run search by clicking button (reveals existing tab)

### Why IntersectionObserver vs Scroll Events?
- More performant (no scroll event listener overhead)
- Browser-optimized intersection detection
- Can trigger before reaching exact bottom (rootMargin)
- Automatic cleanup when element removed from DOM

### Why HTTP Server Instead of Direct Binary Execution?
- Reuses existing Go API server
- Allows multiple consumers (CLI, extension, future web UI)
- Clean separation of concerns
- Easier to debug (can curl endpoints)
- Supports keep-alive connections

## Performance Considerations

### Bundle Size Optimization
- Vite automatically code-splits and tree-shakes
- VS Code CSS variables instead of Tailwind (no CSS overhead)
- No heavy dependencies (just Svelte runtime ~15KB)

### Memory Management
- IntersectionObserver automatically cleaned up in $effect return
- Logs array grows but capped by practical scroll limits
- No memory leaks from event listeners (all cleaned up)

### Server Lifecycle
- Server starts on-demand (first search)
- Stays alive during VS Code session
- Auto-stopped when VS Code closes
- Random port to avoid conflicts

## Security Considerations

### Content Security Policy
Extension uses strict CSP for webviews:
```typescript
const csp = `
  default-src 'none';
  style-src ${webview.cspSource} 'unsafe-inline';
  script-src ${webview.cspSource} 'nonce-${nonce}';
  connect-src http://localhost:${port};
  img-src ${webview.cspSource} data:;
  font-src ${webview.cspSource};
`;
```

### Local-only Server
- Server binds to localhost only
- Random port for security through obscurity
- No external network access
- Uses user's existing credentials/config

## Testing Approach

### Manual Testing Checklist
- [x] Sidebar loads contexts correctly
- [x] Quick filter pills toggle properly
- [x] Search button opens results tab
- [x] Same search reuses existing tab
- [x] Different search opens new tab
- [x] Logs display with correct colors
- [x] Row expansion shows JSON details
- [x] Infinite scroll loads more logs
- [x] Load more button triggers scroll
- [x] Pagination token properly passed
- [x] All logs loaded message shows when done
- [x] Export downloads JSON file
- [x] Copy copies to clipboard
- [x] Recent searches saved and recalled

### Integration Testing
Currently manual. Future automation could:
- Spawn extension host programmatically
- Mock Go server responses
- Test webview message passing
- Verify DOM structure matches expectations

## Lessons Learned

1. **VS Code webview API is stateful**: Can only acquire once. Store and reuse.

2. **Svelte 5 $effect is powerful**: Replaces onMount, onDestroy, and reactive statements with single unified primitive.

3. **Hash-based identity works well**: Simple deterministic approach to tab management without complex state.

4. **Backend pagination requires two pieces**: Token IN (request) and token OUT (response). Easy to forget one.

5. **Type errors reveal architecture**: The `GetEntries()` return type (channel) revealed streaming architecture, leading us to `GetPaginationInfo()`.

6. **Visual feedback matters**: Clickable button performs better than passive text, even when IntersectionObserver works automatically.

7. **CSS custom properties are underrated**: Using VS Code's variables gives perfect theme matching with zero effort.

8. **Build pipeline complexity manageable**: Makefile targets make multi-stage builds (Go + npm) straightforward.

## Future Refactoring Opportunities

1. **Extract API client**: Move from inline in components to shared service
2. **Add state machine**: Manage loading/error/success states more formally
3. **Type safety**: Add runtime validation for API responses (Zod?)
4. **Error boundaries**: Graceful degradation when API fails
5. **Telemetry**: Track which features are used most
6. **Settings UI**: Allow customizing defaults (time range, size, etc.)
7. **Multi-window support**: Currently one server per window, could share
8. **Binary caching**: Avoid restarting server on every search
