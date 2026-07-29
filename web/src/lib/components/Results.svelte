<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { api, type LogEntry, type QueryMeta } from '$lib/api';
  import LogRow from './LogRow.svelte';
  import { get } from 'svelte/store';

  // Props passed from VS Code
  interface Props {
    contextId: string;
    query?: string;
    timeRange?: string;
    level?: string;
    size?: number;
  }

  let { contextId, query = '', timeRange = '1h', level, size = 100 }: Props = $props();

  let logs = $state<LogEntry[]>([]);
  let meta = $state<QueryMeta | null>(null);
  let isLoading = $state(false);
  let error = $state<string | null>(null);
  let loadingMore = $state(false);
  let sentinel: HTMLDivElement;
  let observer: IntersectionObserver | null = null;

  function handleRefresh() {
    executeSearch();
  }

  function handleExport() {
    const json = JSON.stringify(logs, null, 2);
    const blob = new Blob([json], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `logs-${contextId}-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }

  function handleCopy() {
    const text = logs.map(l => `${l.timestamp} ${l.level} ${l.message}`).join('\n');
    navigator.clipboard.writeText(text);
  }

  function handleLoadMore() {
    // Scroll to the sentinel to trigger IntersectionObserver
    if (sentinel) {
      sentinel.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }
  }

  async function executeSearch(append = false) {
    if (!contextId) return;

    if (append) {
      loadingMore = true;
      console.log('[Results] Loading more logs, has nextPageToken:', !!meta?.nextPageToken);
    } else {
      isLoading = true;
    }
    error = null;

    try {
      const response = await api.queryLogs({
        contextId,
        nativeQuery: query || undefined,
        range: { last: timeRange },
        fields: level ? { level } : undefined,
        size: size,
        pageToken: append ? meta?.nextPageToken : undefined,
      });

      console.log('[Results] Received logs:', response.logs.length, 'nextPageToken:', response.meta.nextPageToken);

      // Reverse logs so newest appear first
      const reversedLogs = [...response.logs].reverse();

      if (append) {
        logs = [...logs, ...reversedLogs];
        console.log('[Results] Total logs after append:', logs.length);
      } else {
        logs = reversedLogs;
        console.log('[Results] Initial logs loaded:', logs.length);
      }
      meta = response.meta;
    } catch (e) {
      error = e instanceof Error ? e.message : 'Query failed';
      console.error('[Results] Query failed:', e);
    } finally {
      isLoading = false;
      loadingMore = false;
    }
  }

  function loadMore() {
    console.log('[Results] loadMore called, isLoading:', isLoading, 'loadingMore:', loadingMore, 'hasToken:', !!meta?.nextPageToken);
    // Load older logs when scrolling to bottom
    if (!isLoading && !loadingMore && meta?.nextPageToken) {
      console.log('[Results] Triggering load more');
      executeSearch(true);
    }
  }

  onMount(() => {
    console.log('[Results] Component mounted');
    // Execute initial search - shows newest logs first
    executeSearch();
  });

  $effect(() => {
    // Set up observer when sentinel is available
    if (sentinel && !observer) {
      console.log('[Results] Setting up IntersectionObserver');
      observer = new IntersectionObserver(
        (entries) => {
          console.log('[Results] IntersectionObserver triggered, isIntersecting:', entries[0].isIntersecting);
          if (entries[0].isIntersecting) {
            loadMore();
          }
        },
        {
          threshold: 0.1,
          root: null, // viewport
          rootMargin: '100px' // trigger slightly before reaching bottom
        }
      );
      observer.observe(sentinel);
      console.log('[Results] Sentinel is now being observed');
    }

    return () => {
      if (observer) {
        console.log('[Results] Cleaning up observer');
        observer.disconnect();
        observer = null;
      }
    };
  });

  onDestroy(() => {
    console.log('[Results] Component destroyed');
    observer?.disconnect();
  });
</script>

<div class="results">
  <header class="header">
    <div class="header-left">
      <span class="icon">🔍</span>
      <span class="context">{contextId}</span>
      {#if query}
        <span class="separator">:</span>
        <span class="query">{query}</span>
      {/if}
      <span class="separator">|</span>
      <span class="time-range">{timeRange}</span>
      {#if level}
        <span class="separator">|</span>
        <span class="level-badge">{level}</span>
      {/if}
    </div>
    <div class="header-right">
      <button class="toolbar-btn" onclick={handleRefresh} title="Refresh">⟳</button>
      <button class="toolbar-btn" onclick={handleExport} title="Export">⬇️</button>
      <button class="toolbar-btn" onclick={handleCopy} title="Copy">📋</button>
    </div>
  </header>

  {#if error}
    <div class="error">{error}</div>
  {/if}

  <div class="table-container">
    <table>
      <thead>
        <tr>
          <th class="col-timestamp">Time</th>
          <th class="col-level">Level</th>
          <th class="col-message">Message</th>
        </tr>
      </thead>
      <tbody>
        {#each logs as log, i (log.timestamp + i)}
          <LogRow {log} index={i} />
        {/each}
      </tbody>
    </table>

    <div bind:this={sentinel} class="sentinel"></div>
  </div>

  <footer class="footer">
    <div class="progress-bar">
      <div class="progress-fill" style="width: {meta?.nextPageToken ? '50%' : '100%'}"></div>
    </div>
    <div class="footer-info">
      {#if isLoading}
        <span class="loading-text">Loading...</span>
      {:else if loadingMore}
        <span class="loading-text">Loading older logs...</span>
      {/if}
      {#if meta}
        <span class="result-count">{logs.length} results</span>
        {#if meta.nextPageToken}
          <button class="load-more-btn" onclick={handleLoadMore}>
            Load more logs ↓
          </button>
        {:else if logs.length > 0}
          <span class="all-loaded">• All logs loaded</span>
        {/if}
      {/if}
    </div>
  </footer>
</div>

<style>
  /* Ensure no gaps from parent */
  :global(body) {
    background: var(--vscode-editor-background) !important;
  }

  .results {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    margin: 0;
    padding: 0;
    overflow: hidden;
    background: var(--vscode-editor-background);
    color: var(--vscode-editor-foreground);
    position: relative;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    height: 35px;
    padding: 0 8px;
    border-bottom: 1px solid var(--vscode-panel-border);
    background: var(--vscode-editorGroupHeader-tabsBackground);
    flex-shrink: 0;
  }

  .header-left {
    display: flex;
    gap: 6px;
    align-items: center;
    font-size: 13px;
  }

  .header-right {
    display: flex;
    gap: 4px;
    align-items: center;
  }

  .icon {
    font-size: 14px;
  }

  .context {
    font-weight: 600;
    color: var(--vscode-textLink-foreground);
  }

  .separator {
    color: var(--vscode-descriptionForeground);
    opacity: 0.6;
  }

  .query {
    font-family: var(--vscode-editor-font-family);
    font-size: 12px;
    color: var(--vscode-editor-foreground);
  }

  .time-range {
    font-size: 12px;
    color: var(--vscode-descriptionForeground);
  }

  .level-badge {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    color: var(--vscode-button-foreground);
    background: var(--vscode-button-background);
    padding: 2px 6px;
    border-radius: 2px;
  }

  .toolbar-btn {
    background: none;
    border: none;
    color: var(--vscode-foreground);
    cursor: pointer;
    padding: 4px 6px;
    border-radius: 2px;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .toolbar-btn:hover {
    background: var(--vscode-toolbar-hoverBackground);
  }

  .toolbar-btn:active {
    background: var(--vscode-toolbar-activeBackground);
  }

  .table-container {
    flex: 1;
    overflow: auto;
    min-height: 0;
  }

  table {
    width: 100%;
    margin: 0;
    padding: 0;
    border-collapse: collapse;
    font-family: var(--vscode-editor-font-family);
    font-size: 13px;
    table-layout: auto;
  }

  tbody, thead, tr {
    width: 100%;
  }

  th {
    text-align: left;
    padding: 6px 8px;
    height: 28px;
    background: var(--vscode-editorGroupHeader-tabsBackground);
    position: sticky;
    top: 0;
    border-bottom: 1px solid var(--vscode-panel-border);
    font-weight: 600;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--vscode-descriptionForeground);
    z-index: 1;
    white-space: nowrap;
  }

  .col-timestamp {
    width: 90px;
    min-width: 90px;
  }

  .col-level {
    width: 60px;
    min-width: 60px;
    text-align: left;
  }

  .col-message {
    width: auto;
    min-width: 300px;
  }

  .error {
    padding: 8px;
    color: var(--vscode-errorForeground);
    background: var(--vscode-inputValidation-errorBackground);
    border-bottom: 1px solid var(--vscode-inputValidation-errorBorder);
    font-size: 13px;
  }

  .footer {
    flex-shrink: 0;
    border-top: 1px solid var(--vscode-panel-border);
    background: var(--vscode-editorGroupHeader-tabsBackground);
  }

  .progress-bar {
    height: 2px;
    background: var(--vscode-progressBar-background);
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: var(--vscode-progressBar-background);
    transition: width 0.3s ease;
  }

  .footer-info {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 12px;
    padding: 4px 8px;
    font-size: 12px;
    color: var(--vscode-descriptionForeground);
  }

  .loading-text {
    animation: pulse 1.5s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 0.6; }
    50% { opacity: 1; }
  }

  .result-count {
    font-weight: 500;
  }

  .load-more-btn {
    background: none;
    border: none;
    color: var(--vscode-textLink-foreground);
    cursor: pointer;
    font-size: 12px;
    font-weight: 500;
    padding: 2px 6px;
    border-radius: 2px;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .load-more-btn:hover {
    color: var(--vscode-textLink-activeForeground);
    background: var(--vscode-list-hoverBackground);
  }

  .load-more-btn:active {
    background: var(--vscode-list-activeSelectionBackground);
  }

  .all-loaded {
    color: var(--vscode-descriptionForeground);
    font-size: 11px;
    opacity: 0.8;
  }

  .sentinel {
    height: 20px;
    margin: 10px 0;
  }
</style>
