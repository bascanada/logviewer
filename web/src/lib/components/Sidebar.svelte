<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type Context } from '$lib/api';
  import { contexts, selectedContextId } from '$lib/stores/context';
  import { searchState } from '$lib/stores/search';

  let loading = $state(true);
  let fetchError = $state<string | null>(null);
  let recentSearches = $state<string[]>([]);

  // Acquire VS Code API once and store it
  let vscode: any = null;
  if (typeof acquireVsCodeApi !== 'undefined') {
    vscode = acquireVsCodeApi();
    console.log('[Sidebar] VS Code API acquired');
  }

  const timeRanges = [
    { value: '15m', label: '15 minutes' },
    { value: '1h', label: '1 hour' },
    { value: '24h', label: '24 hours' },
    { value: '7d', label: '7 days' },
  ];

  const sizes = [
    { value: 50, label: '50' },
    { value: 100, label: '100' },
    { value: 250, label: '250' },
    { value: 500, label: '500' },
    { value: 1000, label: '1000' },
  ];

  const levels = ['DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL'];

  onMount(async () => {
    // Load recent searches from localStorage
    try {
      const saved = localStorage.getItem('logviewer-recent-searches');
      if (saved) {
        recentSearches = JSON.parse(saved);
      }
    } catch {}

    // Load contexts
    try {
      const response = await api.getContexts();
      contexts.set(response.contexts);
      if (response.contexts.length > 0 && !$selectedContextId) {
        selectedContextId.set(response.contexts[0].id);
      }
    } catch (e) {
      fetchError = e instanceof Error ? e.message : 'Failed to load contexts';
    } finally {
      loading = false;
    }
  });

  function handleSearch() {
    console.log('[Sidebar] handleSearch called');
    console.log('[Sidebar] Current state:', {
      selectedContextId: $selectedContextId,
      searchState: $searchState
    });

    // Save to recent searches
    const query = $searchState.query.trim();
    if (query && !recentSearches.includes(query)) {
      recentSearches = [query, ...recentSearches.slice(0, 9)];
      localStorage.setItem('logviewer-recent-searches', JSON.stringify(recentSearches));
    }

    // Post message to VS Code to open results panel
    if (vscode) {
      console.log('[Sidebar] VS Code API available, sending message...');
      const message = {
        type: 'search',
        contextId: $selectedContextId,
        query: $searchState.query,
        timeRange: $searchState.timeRange,
        level: $searchState.level,
        size: $searchState.size,
      };
      console.log('[Sidebar] Posting message:', message);
      vscode.postMessage(message);
      console.log('[Sidebar] Message posted');
    } else {
      console.error('[Sidebar] VS Code API is NOT available!');
    }
  }

  function applyRecentSearch(query: string) {
    $searchState.query = query;
    handleSearch();
  }

  function clearRecentSearches() {
    recentSearches = [];
    localStorage.removeItem('logviewer-recent-searches');
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleSearch();
    }
  }
</script>

<div class="sidebar">
  <section class="section">
    <h3>Context</h3>
    {#if loading}
      <span class="loading">Loading...</span>
    {:else if fetchError}
      <span class="error">{fetchError}</span>
    {:else}
      <select bind:value={$selectedContextId}>
        {#each $contexts as ctx}
          <option value={ctx.id}>{ctx.id}</option>
        {/each}
      </select>
      {#if $contexts.find(c => c.id === $selectedContextId)}
        {@const selectedContext = $contexts.find(c => c.id === $selectedContextId)}
        {#if selectedContext?.client || selectedContext?.description}
          <div class="context-info">
            {#if selectedContext.client}
              <span class="info-chip">{selectedContext.client}</span>
            {/if}
            {#if selectedContext.description}
              <span class="info-chip">{selectedContext.description}</span>
            {/if}
          </div>
        {/if}
      {/if}
    {/if}
  </section>

  <section class="section">
    <h3>Quick Filters</h3>
    <div class="filter-pills">
      <button
        class="pill"
        class:active={$searchState.level === 'ERROR'}
        onclick={() => $searchState.level = $searchState.level === 'ERROR' ? null : 'ERROR'}
      >
        ERROR
      </button>
      <button
        class="pill"
        class:active={$searchState.level === 'WARN'}
        onclick={() => $searchState.level = $searchState.level === 'WARN' ? null : 'WARN'}
      >
        WARN
      </button>
      <button
        class="pill"
        class:active={$searchState.level === 'INFO'}
        onclick={() => $searchState.level = $searchState.level === 'INFO' ? null : 'INFO'}
      >
        INFO
      </button>
    </div>
  </section>

  <section class="section">
    <h3>Time Range</h3>
    <select bind:value={$searchState.timeRange}>
      {#each timeRanges as range}
        <option value={range.value}>{range.label}</option>
      {/each}
    </select>
  </section>

  <section class="section">
    <h3>Size</h3>
    <select bind:value={$searchState.size}>
      {#each sizes as size}
        <option value={size.value}>{size.label} results</option>
      {/each}
    </select>
  </section>

  <section class="section">
    <h3>Query</h3>
    <input
      type="text"
      placeholder='service="checkout"'
      bind:value={$searchState.query}
      onkeydown={handleKeydown}
    />

    <button class="primary" onclick={handleSearch}>Search Logs</button>
  </section>

  {#if recentSearches.length > 0}
    <section class="section">
      <div class="section-header">
        <h3>Recent Searches</h3>
        <button class="link" onclick={clearRecentSearches}>Clear</button>
      </div>
      <ul class="recent-list">
        {#each recentSearches as query}
          <li>
            <button class="recent-item" onclick={() => applyRecentSearch(query)}>
              {query}
            </button>
          </li>
        {/each}
      </ul>
    </section>
  {/if}
</div>

<style>
  .sidebar {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 8px;
    width: 100%;
    height: 100%;
    margin: 0;
    overflow-y: auto;
    background: var(--vscode-sideBar-background);
    color: var(--vscode-sideBar-foreground);
    box-sizing: border-box;
    position: relative;
  }

  /* Ensure no gaps from parent */
  :global(body) {
    background: var(--vscode-sideBar-background) !important;
  }

  .section {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  h3 {
    margin: 0;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--vscode-sideBarSectionHeader-foreground);
    opacity: 0.8;
  }

  select, input {
    width: 100%;
    padding: 4px 6px;
    border-radius: 2px;
    background: var(--vscode-input-background);
    color: var(--vscode-input-foreground);
    border: 1px solid var(--vscode-input-border);
    font-family: var(--vscode-font-family);
    font-size: 13px;
  }

  select:focus, input:focus {
    outline: 1px solid var(--vscode-focusBorder);
    border-color: var(--vscode-focusBorder);
  }

  input::placeholder {
    color: var(--vscode-input-placeholderForeground);
  }

  button.primary {
    width: 100%;
    padding: 6px;
    border-radius: 2px;
    background: var(--vscode-button-background);
    color: var(--vscode-button-foreground);
    border: none;
    cursor: pointer;
    font-family: var(--vscode-font-family);
    font-size: 13px;
    font-weight: 500;
  }

  button.primary:hover {
    background: var(--vscode-button-hoverBackground);
  }

  button.primary:active {
    background: var(--vscode-button-hoverBackground);
    opacity: 0.9;
  }

  button.link {
    background: none;
    border: none;
    color: var(--vscode-textLink-foreground);
    cursor: pointer;
    font-size: 11px;
    padding: 0;
  }

  button.link:hover {
    color: var(--vscode-textLink-activeForeground);
    text-decoration: underline;
  }

  .recent-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .recent-item {
    width: 100%;
    text-align: left;
    background: none;
    border: none;
    color: var(--vscode-sideBar-foreground);
    padding: 4px 6px;
    cursor: pointer;
    font-family: var(--vscode-font-family);
    font-size: 13px;
    border-radius: 2px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .recent-item:hover {
    background: var(--vscode-list-hoverBackground);
    color: var(--vscode-list-hoverForeground);
  }

  .recent-item:active {
    background: var(--vscode-list-activeSelectionBackground);
  }

  .context-info {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 4px;
  }

  .info-chip {
    display: inline-block;
    padding: 2px 6px;
    background: var(--vscode-badge-background);
    color: var(--vscode-badge-foreground);
    font-size: 11px;
    border-radius: 2px;
  }

  .filter-pills {
    display: flex;
    gap: 6px;
  }

  .pill {
    flex: 1;
    padding: 4px 8px;
    background: var(--vscode-input-background);
    color: var(--vscode-input-foreground);
    border: 1px solid var(--vscode-input-border);
    border-radius: 2px;
    cursor: pointer;
    font-family: var(--vscode-font-family);
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .pill:hover {
    background: var(--vscode-list-hoverBackground);
  }

  .pill.active {
    background: var(--vscode-button-background);
    color: var(--vscode-button-foreground);
    border-color: var(--vscode-button-background);
  }

  .loading, .error {
    font-size: 13px;
  }

  .error {
    color: var(--vscode-errorForeground);
  }
</style>
