<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { contexts, selectedContextId } from '$lib/stores/context';

  let loading = true;
  let fetchError: string | null = null;

  onMount(async () => {
    try {
      const response = await api.getContexts();
      contexts.set(response.contexts);
      // Auto-select first context
      if (response.contexts.length > 0 && !$selectedContextId) {
        selectedContextId.set(response.contexts[0].id);
      }
    } catch (e) {
      fetchError = e instanceof Error ? e.message : 'Failed to load contexts';
    } finally {
      loading = false;
    }
  });
</script>

<div class="context-selector">
  {#if loading}
    <span class="loading">Loading contexts...</span>
  {:else if fetchError}
    <span class="error">{fetchError}</span>
  {:else}
    <select bind:value={$selectedContextId}>
      {#each $contexts as ctx}
        <option value={ctx.id}>{ctx.id} ({ctx.client})</option>
      {/each}
    </select>
  {/if}
</div>

<style>
  .context-selector {
    padding: 0.5rem;
  }
  select {
    width: 100%;
    padding: 0.5rem;
    border-radius: 4px;
    background: var(--vscode-dropdown-background, #3c3c3c);
    color: var(--vscode-dropdown-foreground, #cccccc);
    border: 1px solid var(--vscode-dropdown-border, #3c3c3c);
  }
  select:focus {
    outline: 1px solid var(--vscode-focusBorder, #007fd4);
  }
  .error {
    color: var(--vscode-errorForeground, #f44336);
  }
  .loading {
    color: var(--vscode-descriptionForeground, #888888);
  }
</style>
