<script lang="ts">
  import GraphVisualizer from '$lib/components/GraphVisualizer.svelte';
  import SearchBar from '$lib/components/SearchBar.svelte';
  import { navigate } from '$lib/router';
  import { onMount, onDestroy } from 'svelte';

  let mounted = false;
  let graphKey = 0;
  let eventSource: EventSource | null = null;

  onMount(() => {
    mounted = true;

    // Listen for vault changes via SSE
    eventSource = new EventSource('/api/v1/events');
    eventSource.addEventListener('graph-updated', () => {
      // Remount GraphVisualizer to reload data
      graphKey++;
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

    if (!node?.id) {
      console.error('Invalid node selected:', node);
      return;
    }

    navigate(`/notes/${node.id}`);
  }
</script>

<svelte:head>
  <title>Mnemosyne</title>
</svelte:head>

<main>
  {#if mounted}
    <div class="chrome-top">
      <div class="brand">
        <span class="brand-mark">M</span>
        <span class="brand-name">mnemosyne</span>
      </div>
      <div class="search-area" role="search" aria-label="Search graph nodes">
        <SearchBar on:select={handleSearchSelect} />
      </div>
    </div>
    {#key graphKey}
      <GraphVisualizer />
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

  .search-area {
    flex: 1;
    max-width: 360px;
  }
</style>
