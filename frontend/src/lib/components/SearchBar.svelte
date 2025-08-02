<script lang="ts">
	import { debounce } from '$lib/utils/debounce';
	import { createEventDispatcher, onDestroy } from 'svelte';

	// Constants
	const MINIMUM_QUERY_LENGTH = 2;
	const SEARCH_DEBOUNCE_MS = 300;
	const BLUR_DELAY_MS = 200;
	const LOADING_SPINNER_DELAY_MS = 100;

	// Types
	interface SearchResult {
		id: string;
		title: string;
		metadata?: {
			type?: string;
			[key: string]: any;
		};
	}

	export let placeholder = 'Search nodes...';

	const dispatch = createEventDispatcher();

	let searchQuery = '';
	let searchResults: SearchResult[] = [];
	let loading = false;
	let showDropdown = false;
	let selectedIndex = -1;
	let searchInput: HTMLInputElement;
	let blurTimeout: number | undefined;
	let loadingTimeout: number | undefined;
	let showSpinner = false;
	let abortController: AbortController | null = null;
	let searchError = '';

	// Debounced search function with proper request cancellation
	const performSearch = debounce(async (query: string) => {
		if (!query || query.length < MINIMUM_QUERY_LENGTH) {
			searchResults = [];
			showDropdown = false;
			searchError = '';
			return;
		}

		// Cancel any in-flight requests
		if (abortController) {
			abortController.abort();
		}
		abortController = new AbortController();

		loading = true;
		searchError = '';

		// Delay showing spinner to avoid flashing on fast connections
		if (loadingTimeout) clearTimeout(loadingTimeout);
		loadingTimeout = window.setTimeout(() => {
			if (loading) showSpinner = true;
		}, LOADING_SPINNER_DELAY_MS);

		try {
			// Using correct API endpoint - proxied through Vite config
			const response = await fetch(`/api/v1/nodes/search?q=${encodeURIComponent(query)}`, {
				signal: abortController.signal
			});

			if (response.ok) {
				const data = await response.json();
				searchResults = data.nodes || [];
				showDropdown = searchResults.length > 0 || searchError !== '';
				selectedIndex = -1;
			} else {
				searchError = 'Search service unavailable';
				searchResults = [];
				showDropdown = true;
			}
		} catch (error) {
			if (error instanceof Error && error.name === 'AbortError') {
				// Request was cancelled, ignore
				return;
			}
			searchError = 'Network error. Please check your connection.';
			searchResults = [];
			showDropdown = true;
		} finally {
			loading = false;
			showSpinner = false;
			if (loadingTimeout) {
				clearTimeout(loadingTimeout);
				loadingTimeout = undefined;
			}
		}
	}, SEARCH_DEBOUNCE_MS);

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

	function selectResult(result: SearchResult) {
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
		if (blurTimeout) clearTimeout(blurTimeout);
		blurTimeout = window.setTimeout(() => {
			closeDropdown();
		}, BLUR_DELAY_MS);
	}

	// Cleanup on component destroy
	onDestroy(() => {
		if (blurTimeout) clearTimeout(blurTimeout);
		if (loadingTimeout) clearTimeout(loadingTimeout);
		if (abortController) abortController.abort();
	});
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
			role="combobox"
			aria-expanded={showDropdown}
			aria-controls="search-dropdown"
			aria-activedescendant={selectedIndex >= 0 ? `result-${selectedIndex}` : null}
			aria-label="Search nodes"
		/>
		{#if showSpinner}
			<div class="loading-spinner"></div>
		{/if}
	</div>

	<!-- Screen reader announcement -->
	<div class="sr-only" role="status" aria-live="polite">
		{#if loading}
			Searching...
		{:else if searchError}
			{searchError}
		{:else if showDropdown && searchResults.length > 0}
			{searchResults.length} {searchResults.length === 1 ? 'result' : 'results'} found
		{:else if showDropdown && searchResults.length === 0}
			No results found
		{/if}
	</div>

	{#if showDropdown}
		<div class="search-dropdown" role="listbox" aria-label="Search results">
			{#if searchError}
				<div class="search-error">
					<svg class="error-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
						<circle cx="12" cy="12" r="10"></circle>
						<line x1="12" y1="8" x2="12" y2="12"></line>
						<line x1="12" y1="16" x2="12.01" y2="16"></line>
					</svg>
					<span>{searchError}</span>
				</div>
			{:else}
				{#each searchResults as result, index}
					<button
						class="search-result"
						class:selected={index === selectedIndex}
						on:click={() => selectResult(result)}
						on:mouseenter={() => selectedIndex = index}
						role="option"
						aria-selected={index === selectedIndex}
						id="result-{index}"
					>
						<!-- Svelte automatically escapes text content, preventing XSS -->
						<div class="result-title">{result.title}</div>
						{#if result.metadata?.type}
							<span class="result-type">{result.metadata.type}</span>
						{/if}
					</button>
				{/each}
			{/if}
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

	.search-error {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 1rem;
		color: var(--color-text-secondary);
		font-size: 0.9rem;
	}

	.error-icon {
		color: #f39c12;
		flex-shrink: 0;
	}

	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border-width: 0;
	}
</style>
