<script lang="ts">
  import { route, navigate } from '$lib/router';
  import type { Route } from '$lib/router';
  import Toast from '$lib/components/Toast.svelte';
  import ErrorBoundary from '$lib/components/ErrorBoundary.svelte';
  import { toast } from '$lib/stores/toast';
  import GraphPage from '$lib/pages/GraphPage.svelte';
  import GraphListPage from '$lib/pages/GraphListPage.svelte';
  import { onMount, onDestroy } from 'svelte';

  interface GraphEntry {
    id: number;
    vault_name: string;
    name: string;
    root_path: string;
  }

  let graphs: GraphEntry[] = [];
  let homeGraph = '';
  let loaded = false;
  let eventSource: EventSource | null = null;

  function resolveGraphId(r: Route, graphList: GraphEntry[]): number | null {
    if (r.type !== 'graph') return null;
    const match = graphList.find(g => g.vault_name === r.vaultName && g.root_path === r.graphPath);
    return match ? match.id : null;
  }

  function graphUrl(g: GraphEntry): string {
    return `/${encodeURIComponent(g.vault_name)}/${encodeURIComponent(g.root_path)}`;
  }

  async function fetchGraphs() {
    try {
      const res = await fetch('/api/v1/graphs');
      if (!res.ok) throw new Error(`Server error: ${res.status}`);
      const data = await res.json();
      graphs = data.graphs || [];
      if (data.home_graph) homeGraph = data.home_graph;
    } catch (e) {
      console.error('Failed to fetch graphs:', e);
      toast.error('Unable to reach the server. Please check that the backend is running.');
    }
  }

  onMount(async () => {
    await fetchGraphs();
    loaded = true;

    if ($route.type === 'home' && graphs.length > 0) {
      if (homeGraph) {
        navigate('/' + homeGraph);
      } else {
        navigate(graphUrl(graphs[0]));
      }
    }

    // Listen for graph list changes (GRAPH.yaml added/removed)
    eventSource = new EventSource('/api/v1/events');
    eventSource.onerror = () => {
      if (eventSource?.readyState === EventSource.CLOSED) {
        toast.error('Lost connection to the server. Real-time updates are unavailable.');
      }
    };
    eventSource.addEventListener('graphs-changed', async () => {
      const oldGraphId = resolveGraphId($route, graphs);
      await fetchGraphs();
      const newGraphId = resolveGraphId($route, graphs);

      // Current graph was deleted — redirect to home graph, first available, or home
      if (oldGraphId != null && newGraphId == null) {
        if (homeGraph) {
          navigate('/' + homeGraph);
        } else if (graphs.length > 0) {
          navigate(graphUrl(graphs[0]));
        } else {
          navigate('/');
        }
      }
    });
  });

  onDestroy(() => {
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }
  });

  $: graphId = resolveGraphId($route, graphs);
</script>

<Toast />
<ErrorBoundary>
  {#if loaded}
    {#if $route.type === 'graph' && graphId != null}
      <GraphPage {graphId} {graphs} {graphUrl} />
    {:else}
      <GraphListPage {graphs} {graphUrl} />
    {/if}
  {/if}
</ErrorBoundary>
