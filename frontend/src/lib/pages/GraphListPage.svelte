<script lang="ts">
  import { navigate } from '$lib/router';

  export let graphs: { id: number; vault_name: string; name: string; root_path: string; node_count?: number }[] = [];
  export let graphUrl: (g: any) => string;

  function selectGraph(g: any) {
    navigate(graphUrl(g));
  }

  // Group graphs by vault
  $: grouped = graphs.reduce<Record<string, typeof graphs>>((acc, g) => {
    const key = g.vault_name || 'Unknown';
    if (!acc[key]) acc[key] = [];
    acc[key].push(g);
    return acc;
  }, {});
</script>

<svelte:head>
  <title>Mnemosyne - Graphs</title>
</svelte:head>

<div class="graph-list">
  <header>
    <div class="brand">
      <span class="brand-mark">M</span>
      <h1>mnemosyne</h1>
    </div>
  </header>

  <main>
    {#if graphs.length === 0}
      <p class="status">No graphs found. Add a GRAPH.yaml file to a vault directory.</p>
    {:else}
      {#each Object.entries(grouped) as [vaultName, vaultGraphs]}
        <section class="vault-group">
          <h2>{vaultName}</h2>
          <div class="graph-cards">
            {#each vaultGraphs as g}
              <button class="graph-card" on:click={() => selectGraph(g)}>
                <div class="graph-name">{g.name}</div>
                <div class="graph-meta">
                  {#if g.root_path}
                    <span class="graph-path">{g.root_path}/</span>
                  {/if}
                  {#if g.node_count}
                    <span class="graph-count">{g.node_count} nodes</span>
                  {/if}
                </div>
              </button>
            {/each}
          </div>
        </section>
      {/each}
    {/if}
  </main>
</div>

<style>
  .graph-list {
    min-height: 100vh;
    background: var(--color-background);
    color: var(--color-text);
  }

  header {
    padding: 2rem;
    border-bottom: 1px solid var(--color-border);
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .brand-mark {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-accent-dim);
    color: var(--color-accent);
    font-weight: 500;
    font-size: 16px;
    border-radius: var(--radius-sm);
  }

  h1 {
    font-size: 16px;
    font-weight: 400;
    color: var(--color-text-secondary);
    letter-spacing: 0.04em;
    text-transform: lowercase;
    margin: 0;
  }

  main {
    padding: 2rem;
    max-width: 720px;
    margin: 0 auto;
  }

  .status {
    color: var(--color-text-secondary);
    text-align: center;
    padding: 3rem 0;
  }

  .vault-group {
    margin-bottom: 2rem;
  }

  h2 {
    font-size: 12px;
    font-weight: 500;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    margin-bottom: 0.75rem;
  }

  .graph-cards {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .graph-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 14px 16px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
    transition: all 0.15s;
    text-align: left;
    color: var(--color-text);
    font-family: var(--font-body);
    width: 100%;
  }

  .graph-card:hover {
    border-color: var(--color-border-focus);
    background: var(--color-surface-raised);
  }

  .graph-name {
    font-size: 14px;
    font-weight: 500;
  }

  .graph-meta {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 12px;
    color: var(--color-text-muted);
  }

  .graph-path {
    font-family: var(--font-mono);
    font-size: 11px;
  }
</style>
