<script lang="ts">
	import { debounce } from '$lib/utils/debounce';
	import { createEventDispatcher } from 'svelte';
	
	export let placeholder = 'Search nodes...';
	
	const dispatch = createEventDispatcher();
	
	let searchQuery = '';
	let searchResults: any[] = [];
	let loading = false;
	let showDropdown = false;
	let selectedIndex = -1;
	let searchInput: HTMLInputElement;
	
	// Debounced search function
	const performSearch = debounce(async (query: string) => {
		if (!query || query.length < 2) {
			searchResults = [];
			showDropdown = false;
			return;
		}
		
		loading = true;
		
		try {
			const response = await fetch(`/api/v1/search?q=${encodeURIComponent(query)}`);
			if (response.ok) {
				const data = await response.json();
				searchResults = data.nodes || [];
				showDropdown = searchResults.length > 0;
				selectedIndex = -1;
			} else {
				console.error('Search failed:', response.statusText);
				searchResults = [];
				// Don't show error toast for search - just silently fail
			}
		} catch (error) {
			console.error('Search failed:', error);
			searchResults = [];
			// Don't show error toast for search - just silently fail
		} finally {
			loading = false;
		}
	}, 300);
	
	function handleInput() {
		performSearch(searchQuery);
	}
	
	function handleKeydown(event: KeyboardEvent) {
		if (!showDropdown || searchResults.length === 0) return;
		
		switch (event.key) {
			case 'ArrowDown':
				event.preventDefault();
				selectedIndex = Math.min(selectedIndex + 1, searchResults.length - 1);
				break;
			case 'ArrowUp':
				event.preventDefault();
				selectedIndex = Math.max(selectedIndex - 1, -1);
				break;
			case 'Enter':
				event.preventDefault();
				if (selectedIndex >= 0) {
					selectResult(searchResults[selectedIndex]);
				}
				break;
			case 'Escape':
				event.preventDefault();
				closeDropdown();
				break;
		}
	}
	
	function selectResult(result: any) {
		dispatch('select', result);
		searchQuery = '';
		searchResults = [];
		showDropdown = false;
		selectedIndex = -1;
	}
	
	function closeDropdown() {
		showDropdown = false;
		selectedIndex = -1;
	}
	
	function handleBlur() {
		// Delay to allow click events on results
		setTimeout(() => {
			closeDropdown();
		}, 200);
	}
</script>

<div class="search-container">
	<div class="search-input-wrapper">
		<svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
			<circle cx="11" cy="11" r="8"></circle>
			<path d="m21 21-4.35-4.35"></path>
		</svg>
		<input
			bind:this={searchInput}
			bind:value={searchQuery}
			on:input={handleInput}
			on:keydown={handleKeydown}
			on:blur={handleBlur}
			type="text"
			{placeholder}
			class="search-input"
			autocomplete="off"
		/>
		{#if loading}
			<div class="loading-spinner"></div>
		{/if}
	</div>
	
	{#if showDropdown}
		<div class="search-dropdown">
			{#each searchResults as result, index}
				<button
					class="search-result"
					class:selected={index === selectedIndex}
					on:click={() => selectResult(result)}
					on:mouseenter={() => selectedIndex = index}
				>
					<div class="result-title">{result.title}</div>
					{#if result.metadata?.type}
						<span class="result-type">{result.metadata.type}</span>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.search-container {
		position: relative;
		width: 300px;
	}
	
	.search-input-wrapper {
		position: relative;
		display: flex;
		align-items: center;
	}
	
	.search-icon {
		position: absolute;
		left: 12px;
		color: var(--color-text-secondary);
		pointer-events: none;
	}
	
	.search-input {
		width: 100%;
		padding: 0.75rem 1rem 0.75rem 2.5rem;
		background-color: var(--color-surface);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 8px;
		color: var(--color-text);
		font-size: 0.9rem;
		transition: all 0.2s;
	}
	
	.search-input:focus {
		outline: none;
		border-color: var(--color-primary);
		box-shadow: 0 0 0 2px rgba(58, 123, 213, 0.2);
	}
	
	.search-input::placeholder {
		color: var(--color-text-secondary);
	}
	
	.loading-spinner {
		position: absolute;
		right: 12px;
		width: 16px;
		height: 16px;
		border: 2px solid rgba(255, 255, 255, 0.1);
		border-top-color: var(--color-primary);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}
	
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
	
	.search-dropdown {
		position: absolute;
		top: calc(100% + 4px);
		left: 0;
		right: 0;
		background-color: var(--color-surface);
		border: 1px solid rgba(255, 255, 255, 0.1);
		border-radius: 8px;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
		max-height: 300px;
		overflow-y: auto;
		z-index: 1000;
	}
	
	.search-result {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		padding: 0.75rem 1rem;
		background: none;
		border: none;
		color: var(--color-text);
		text-align: left;
		cursor: pointer;
		transition: background-color 0.2s;
	}
	
	.search-result:hover,
	.search-result.selected {
		background-color: rgba(58, 123, 213, 0.1);
	}
	
	.search-result:not(:last-child) {
		border-bottom: 1px solid rgba(255, 255, 255, 0.05);
	}
	
	.result-title {
		font-size: 0.9rem;
		font-weight: 500;
	}
	
	.result-type {
		font-size: 0.75rem;
		color: var(--color-text-secondary);
		background-color: rgba(255, 255, 255, 0.05);
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		text-transform: capitalize;
	}
</style>