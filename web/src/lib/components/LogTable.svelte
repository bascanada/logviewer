<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { api } from '$lib/api';
  import { logs, meta, isLoading, error } from '$lib/stores/logs';
  import { selectedContextId } from '$lib/stores/context';
  import { searchState } from '$lib/stores/search';
  import LogRow from './LogRow.svelte';
  import { get } from 'svelte/store';

  let sentinel: HTMLDivElement;
  let observer: IntersectionObserver;

  export async function executeSearch(append = false) {
    const contextId = get(selectedContextId);
    if (!contextId) return;

    isLoading.set(true);
    error.set(null);

    try {
      const currentSearch = get(searchState);
      const currentMeta = get(meta);

      const response = await api.queryLogs({
        contextId,
        nativeQuery: currentSearch.query || undefined,
        range: { last: currentSearch.timeRange },
        fields: currentSearch.level ? { level: currentSearch.level } : undefined,
        size: 100,
        pageToken: append ? currentMeta?.nextPageToken : undefined,
      });

      if (append) {
        logs.update((current) => [...current, ...response.logs]);
      } else {
        logs.set(response.logs);
      }
      meta.set(response.meta);
    } catch (e) {
      error.set(e instanceof Error ? e.message : 'Query failed');
    } finally {
      isLoading.set(false);
    }
  }

  function loadMore() {
    const loading = get(isLoading);
    const currentMeta = get(meta);
    if (!loading && currentMeta?.nextPageToken) {
      executeSearch(true);
    }
  }

  onMount(() => {
    observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          loadMore();
        }
      },
      { threshold: 0.1 }
    );

    if (sentinel) {
      observer.observe(sentinel);
    }
  });

  onDestroy(() => {
    observer?.disconnect();
  });
</script>

<div class="log-table">
  {#if $error}
    <div class="error">{$error}</div>
  {/if}

  <table>
    <thead>
      <tr>
        <th class="timestamp">Timestamp</th>
        <th class="level">Level</th>
        <th class="message">Message</th>
      </tr>
    </thead>
    <tbody>
      {#each $logs as log, i (log.timestamp + i)}
        <LogRow {log} />
      {/each}
    </tbody>
  </table>

  {#if $isLoading}
    <div class="loading">Loading...</div>
  {/if}

  <!-- Sentinel for infinite scroll -->
  <div bind:this={sentinel} class="sentinel"></div>

  {#if $meta}
    <div class="meta">
      {$meta.resultCount} results in {$meta.queryTime}
    </div>
  {/if}
</div>

<style>
  .log-table {
    flex: 1;
    overflow: auto;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-family: monospace;
    font-size: 12px;
  }
  th {
    text-align: left;
    padding: 0.5rem;
    background: var(--vscode-editor-background, #1e1e1e);
    position: sticky;
    top: 0;
    border-bottom: 1px solid var(--vscode-panel-border, #333333);
  }
  .timestamp {
    width: 180px;
  }
  .level {
    width: 60px;
  }
  .error {
    padding: 1rem;
    color: var(--vscode-errorForeground, #f44336);
  }
  .loading,
  .meta {
    padding: 0.5rem;
    text-align: center;
    color: var(--vscode-descriptionForeground, #888888);
  }
  .sentinel {
    height: 1px;
  }
</style>
