<script lang="ts">
	import GraphVisualizer from '$lib/components/GraphVisualizer.svelte';
	import { onMount } from 'svelte';
	
	let mounted = false;
	
	onMount(() => {
		mounted = true;
	});
</script>

<svelte:head>
	<title>Mnemosyne - Knowledge Graph</title>
</svelte:head>

<main>
	{#if mounted}
		<GraphVisualizer />
	{/if}
</main>

<style>
	main {
		height: 100vh;
		width: 100vw;
		position: relative;
		overflow: hidden;
	}
</style>