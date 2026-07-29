<script lang="ts">
  import ContextSelector from '$lib/components/ContextSelector.svelte';
  import SearchBar from '$lib/components/SearchBar.svelte';
  import LogTable from '$lib/components/LogTable.svelte';

  let logTable: LogTable;

  function handleSearch() {
    logTable?.executeSearch();
  }
</script>

<main>
  <header>
    <h1>LogViewer</h1>
    <ContextSelector />
  </header>

  <SearchBar onSearch={handleSearch} />

  <LogTable bind:this={logTable} />
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    background: var(--vscode-editor-background, #1e1e1e);
    color: var(--vscode-editor-foreground, #d4d4d4);
    font-family: var(--vscode-font-family, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif);
  }
  :global(*) {
    box-sizing: border-box;
  }
  main {
    display: flex;
    flex-direction: column;
    height: 100vh;
  }
  header {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.5rem;
    border-bottom: 1px solid var(--vscode-panel-border, #333333);
  }
  h1 {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
  }
</style>
