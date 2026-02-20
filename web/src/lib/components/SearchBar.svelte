<script lang="ts">
  import { searchState } from '$lib/stores/search';

  interface Props {
    onSearch?: () => void;
  }

  let { onSearch }: Props = $props();

  const timeRanges = [
    { value: '15m', label: '15 minutes' },
    { value: '1h', label: '1 hour' },
    { value: '24h', label: '24 hours' },
    { value: '7d', label: '7 days' },
  ];

  const levels = ['DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL'];

  function handleSubmit() {
    onSearch?.();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleSubmit();
    }
  }
</script>

<div class="search-bar">
  <input
    type="text"
    placeholder="Search query (e.g., level=ERROR or native query)"
    bind:value={$searchState.query}
    onkeydown={handleKeydown}
  />

  <select bind:value={$searchState.timeRange}>
    {#each timeRanges as range}
      <option value={range.value}>{range.label}</option>
    {/each}
  </select>

  <select bind:value={$searchState.level}>
    <option value={null}>All Levels</option>
    {#each levels as level}
      <option value={level}>{level}</option>
    {/each}
  </select>

  <button onclick={handleSubmit}>Search</button>
</div>

<style>
  .search-bar {
    display: flex;
    gap: 0.5rem;
    padding: 0.5rem;
    border-bottom: 1px solid var(--vscode-panel-border, #333333);
  }
  input {
    flex: 1;
    padding: 0.5rem;
    border-radius: 4px;
    background: var(--vscode-input-background, #3c3c3c);
    color: var(--vscode-input-foreground, #cccccc);
    border: 1px solid var(--vscode-input-border, #3c3c3c);
  }
  input:focus {
    outline: 1px solid var(--vscode-focusBorder, #007fd4);
  }
  input::placeholder {
    color: var(--vscode-input-placeholderForeground, #888888);
  }
  select {
    padding: 0.5rem;
    border-radius: 4px;
    background: var(--vscode-dropdown-background, #3c3c3c);
    color: var(--vscode-dropdown-foreground, #cccccc);
    border: 1px solid var(--vscode-dropdown-border, #3c3c3c);
  }
  select:focus {
    outline: 1px solid var(--vscode-focusBorder, #007fd4);
  }
  button {
    padding: 0.5rem 1rem;
    border-radius: 4px;
    background: var(--vscode-button-background, #0078d4);
    color: var(--vscode-button-foreground, #ffffff);
    border: none;
    cursor: pointer;
  }
  button:hover {
    background: var(--vscode-button-hoverBackground, #026ec1);
  }
</style>
