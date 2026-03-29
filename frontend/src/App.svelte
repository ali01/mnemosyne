<script lang="ts">
  import { route } from '$lib/router';
  import Toast from '$lib/components/Toast.svelte';
  import ErrorBoundary from '$lib/components/ErrorBoundary.svelte';
  import GraphPage from '$lib/pages/GraphPage.svelte';
  import NotePage from '$lib/pages/NotePage.svelte';
</script>

<Toast />
<ErrorBoundary>
  {#if $route.path === '/notes/:id'}
    <NotePage id={$route.params.id} />
  {:else}
    <GraphPage />
  {/if}
</ErrorBoundary>
