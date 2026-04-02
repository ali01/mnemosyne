<script lang="ts">
  import GraphVisualizer from '$lib/components/GraphVisualizer.svelte';
  import SearchBar from '$lib/components/SearchBar.svelte';
  import { navigate } from '$lib/router';
  import { openInObsidian } from '$lib/utils/obsidian';
  import { onMount, onDestroy } from 'svelte';

  export let graphId: number;
  export let graphs: { id: number; vault_name: string; name: string; root_path: string }[] = [];
  export let graphUrl: (g: any) => string;

  let mounted = false;
  let graphKey = 0;
  let eventSource: EventSource | null = null;

  $: currentGraph = graphs.find(g => g.id === graphId);
  $: graphName = currentGraph ? currentGraph.name : '';

  onMount(() => {
    mounted = true;

    eventSource = new EventSource('/api/v1/events');
    eventSource.addEventListener('graph-updated', (e) => {
      try {
        const data = JSON.parse(e.data);
        const affected = data.graphIds || data.GraphIDs || [];
        if (affected.includes(graphId)) {
          graphKey++;
        }
      } catch {
        graphKey++;
      }
    });
  });

  onDestroy(() => {
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }
  });

  function handleSearchSelect(event: CustomEvent) {
    const node = event.detail;
    if (!node?.file_path || !currentGraph) return;
    openInObsidian(currentGraph.vault_name, node.file_path);
  }

  function switchGraph(event: Event) {
    const select = event.currentTarget as HTMLSelectElement;
    const id = parseInt(select.value);
    const g = graphs.find(gr => gr.id === id);
    if (g) navigate(graphUrl(g));
  }
</script>

<svelte:head>
  <title>{graphName || 'Graph'} - Mnemosyne</title>
</svelte:head>

<main>
  {#if mounted}
    <div class="chrome-top">
      <div class="brand">
        <span class="brand-mark">M</span>
        <span class="brand-name">mnemosyne</span>
      </div>
      {#if graphs.length > 1}
        <select class="graph-selector" value={graphId} on:change={switchGraph}>
          {#each graphs as g}
            <option value={g.id}>{g.vault_name} / {g.name}</option>
          {/each}
        </select>
      {:else if graphName}
        <span class="graph-label">{graphName}</span>
      {/if}
      <div class="search-area" role="search" aria-label="Search graph nodes">
        <SearchBar graphId={String(graphId)} on:select={handleSearchSelect} />
      </div>
    </div>
    {#key `${graphId}-${graphKey}`}
      <GraphVisualizer graphId={String(graphId)} vaultName={currentGraph?.vault_name || ''} />
    {/key}
  {/if}
</main>

<style>
  main {
    height: 100vh;
    width: 100vw;
    position: relative;
    overflow: hidden;
    background: var(--color-void);
  }

  .chrome-top {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 14px 20px;
    pointer-events: none;
    background: linear-gradient(
      to bottom,
      var(--color-void) 0%,
      rgba(10, 10, 15, 0.8) 60%,
      transparent 100%
    );
  }

  .chrome-top > * {
    pointer-events: auto;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }

  .brand-mark {
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-accent-dim);
    color: var(--color-accent);
    font-family: var(--font-body);
    font-weight: 500;
    font-size: 14px;
    border-radius: var(--radius-sm);
    letter-spacing: 0.02em;
  }

  .brand-name {
    font-family: var(--font-body);
    font-size: 13px;
    font-weight: 400;
    color: var(--color-text-secondary);
    letter-spacing: 0.04em;
    text-transform: lowercase;
  }

  .graph-selector {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-text-secondary);
    font-family: var(--font-body);
    font-size: 12px;
    padding: 5px 8px;
    cursor: pointer;
  }

  .graph-selector:focus {
    outline: none;
    border-color: var(--color-border-focus);
  }

  .graph-label {
    font-size: 12px;
    color: var(--color-text-muted);
    font-family: var(--font-body);
  }

  .search-area {
    flex: 1;
    max-width: 360px;
  }
</style>
