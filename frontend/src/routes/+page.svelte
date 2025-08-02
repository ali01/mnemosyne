<script lang="ts">
	import GraphVisualizer from '$lib/components/GraphVisualizer.svelte';
	import SearchBar from '$lib/components/SearchBar.svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	let mounted = false;

	onMount(() => {
		mounted = true;
	});

	function handleSearchSelect(event: CustomEvent) {
		const node = event.detail;

		// Validate node ID before navigation
		if (!node?.id) {
			console.error('Invalid node selected:', node);
			return;
		}

		// Navigate to the selected node's content with error handling
		try {
			goto(`/notes/${node.id}`);
		} catch (error) {
			console.error('Navigation failed:', error);
		}
	}
</script>

<svelte:head>
	<title>Mnemosyne - Knowledge Graph</title>
</svelte:head>

<main>
	{#if mounted}
		<div class="search-overlay" role="search" aria-label="Search graph nodes">
			<SearchBar on:select={handleSearchSelect} />
		</div>
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

	:root {
		--spacing-md: 20px;
		--z-index-overlay: 100;
	}

	.search-overlay {
		position: absolute;
		top: var(--spacing-md);
		left: var(--spacing-md);
		z-index: var(--z-index-overlay);
	}
</style>
