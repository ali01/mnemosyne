<script lang="ts">
	import GraphVisualizer from '$lib/components/GraphVisualizer.svelte';
	import SearchBar from '$lib/components/SearchBar.svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	
	let mounted = false;
	let graphVisualizerComponent: GraphVisualizer;
	
	onMount(() => {
		mounted = true;
	});
	
	function handleSearchSelect(event: CustomEvent) {
		const node = event.detail;
		// Navigate to the selected node's content
		goto(`/notes/${node.id}`);
		
		// TODO: Add highlighting/focus on the node in the graph
		// This would require exposing a method on GraphVisualizer to focus on a node
	}
</script>

<svelte:head>
	<title>Mnemosyne - Knowledge Graph</title>
</svelte:head>

<main>
	{#if mounted}
		<div class="search-overlay">
			<SearchBar on:select={handleSearchSelect} />
		</div>
		<GraphVisualizer bind:this={graphVisualizerComponent} />
	{/if}
</main>

<style>
	main {
		height: 100vh;
		width: 100vw;
		position: relative;
		overflow: hidden;
	}
	
	.search-overlay {
		position: absolute;
		top: 20px;
		left: 20px;
		z-index: 100;
	}
</style>