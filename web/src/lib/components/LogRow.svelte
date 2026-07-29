<script lang="ts">
  import type { LogEntry } from '$lib/api';

  interface Props {
    log: LogEntry;
    index: number;
  }
  let { log, index }: Props = $props();
  let expanded = $state(false);

  function getLevelClass(level: string): string {
    switch (level?.toUpperCase()) {
      case 'ERROR':
      case 'FATAL':
        return 'level-error';
      case 'WARN':
        return 'level-warn';
      case 'DEBUG':
        return 'level-debug';
      default:
        return 'level-info';
    }
  }

  function formatTimestamp(ts: string): string {
    try {
      const date = new Date(ts);
      const hours = String(date.getHours()).padStart(2, '0');
      const minutes = String(date.getMinutes()).padStart(2, '0');
      const seconds = String(date.getSeconds()).padStart(2, '0');
      return `${hours}:${minutes}:${seconds}`;
    } catch {
      return ts;
    }
  }

  function toggleExpanded() {
    expanded = !expanded;
  }

  function copyToClipboard() {
    navigator.clipboard.writeText(JSON.stringify(log.fields, null, 2));
  }
</script>

<tr class="log-row" class:expanded class:even={index % 2 === 0} onclick={toggleExpanded}>
  <td class="timestamp">{formatTimestamp(log.timestamp)}</td>
  <td class="level">
    <span class="level-badge {getLevelClass(log.level)}">{log.level}</span>
  </td>
  <td class="message">
    {log.message}
    {#if expanded}
      <span class="expand-icon">▼</span>
    {:else}
      <span class="expand-icon">▶</span>
    {/if}
  </td>
</tr>

{#if expanded}
  <tr class="detail-row" class:even={index % 2 === 0}>
    <td colspan="3">
      <div class="detail-container">
        <div class="detail-header">
          <span class="detail-title">Full Log Entry</span>
          <button class="detail-btn" onclick={copyToClipboard}>Copy</button>
          <button class="detail-btn" onclick={toggleExpanded}>Close</button>
        </div>
        <pre class="detail-json">{JSON.stringify(log.fields, null, 2)}</pre>
      </div>
    </td>
  </tr>
{/if}

<style>
  .log-row {
    cursor: pointer;
    height: 24px;
  }

  .log-row.even {
    background: var(--vscode-editor-background);
  }

  .log-row:hover {
    background: var(--vscode-list-hoverBackground);
  }

  .log-row.expanded {
    background: var(--vscode-list-activeSelectionBackground);
  }

  td {
    padding: 4px 8px;
    border-bottom: 1px solid var(--vscode-panel-border);
    vertical-align: middle;
    font-family: var(--vscode-editor-font-family);
    font-size: 13px;
    text-align: left;
  }

  .timestamp {
    white-space: nowrap;
    color: var(--vscode-descriptionForeground);
    font-family: var(--vscode-editor-font-family);
    text-align: left;
  }

  .level {
    text-align: left;
  }

  .level-badge {
    display: inline-block;
    padding: 2px 6px;
    border-radius: 2px;
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .level-error {
    color: var(--vscode-errorForeground);
    background: color-mix(in srgb, var(--vscode-errorForeground) 15%, transparent);
  }

  .level-warn {
    color: var(--vscode-editorWarning-foreground);
    background: color-mix(in srgb, var(--vscode-editorWarning-foreground) 15%, transparent);
  }

  .level-info {
    color: var(--vscode-editorInfo-foreground);
    background: color-mix(in srgb, var(--vscode-editorInfo-foreground) 15%, transparent);
  }

  .level-debug {
    color: var(--vscode-descriptionForeground);
    background: color-mix(in srgb, var(--vscode-descriptionForeground) 15%, transparent);
  }

  .message {
    white-space: nowrap;
    text-align: left;
    position: relative;
    min-width: 300px;
  }

  .expand-icon {
    float: right;
    margin-left: 8px;
    font-size: 10px;
    color: var(--vscode-descriptionForeground);
  }

  .detail-row td {
    padding: 0;
    background: var(--vscode-textCodeBlock-background);
  }

  .detail-row.even td {
    background: var(--vscode-textCodeBlock-background);
  }

  .detail-container {
    padding: 8px;
  }

  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 4px 8px;
    margin-bottom: 8px;
    background: var(--vscode-editorGroupHeader-tabsBackground);
    border-radius: 2px;
  }

  .detail-title {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--vscode-descriptionForeground);
  }

  .detail-btn {
    background: none;
    border: 1px solid var(--vscode-input-border);
    color: var(--vscode-button-foreground);
    cursor: pointer;
    padding: 2px 8px;
    margin-left: 4px;
    border-radius: 2px;
    font-size: 11px;
  }

  .detail-btn:hover {
    background: var(--vscode-button-hoverBackground);
  }

  .detail-json {
    margin: 0;
    padding: 8px;
    background: var(--vscode-editor-background);
    border: 1px solid var(--vscode-panel-border);
    border-radius: 2px;
    overflow-x: auto;
    font-size: 12px;
    font-family: var(--vscode-editor-font-family);
    white-space: pre-wrap;
    word-break: break-word;
    color: var(--vscode-editor-foreground);
  }
</style>
