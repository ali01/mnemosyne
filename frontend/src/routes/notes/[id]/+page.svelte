<script lang="ts">
	import { page } from '$app/stores';
	import { onMount, onDestroy } from 'svelte';
	import { marked } from 'marked';
	import { goto } from '$app/navigation';
	import { wikilinkExtension } from '$lib/utils/wikilink-renderer';
	import { fetchWithRetry } from '$lib/stores/graph';
	import DOMPurify from 'isomorphic-dompurify';

	// Constants
	const ROUTES = {
		GRAPH: '/',
		NODE_CONTENT: '/api/v1/nodes/:id/content'
	};

	const LOADING_MESSAGES = {
		NOTE: 'Loading note...',
		NOT_FOUND: 'Note not found',
		INVALID_ID: 'Invalid note ID',
		LOAD_ERROR: 'Failed to load note content. Please try again.'
	};

	const NODE_ID_PATTERN = /^[a-zA-Z0-9_-]+$/;

	// TypeScript interface for API response
	interface NodeContentResponse {
		title: string;
		content: string;
	}

	let nodeContent = '';
	let nodeTitle = '';
	let loading = true;
	let error = '';
	let abortController: AbortController | null = null;

	// Configure marked with WikiLink extension at module level
	marked.use(wikilinkExtension);

	onMount(async () => {
		const nodeId = $page.params.id;

		// Validate route parameter
		if (!nodeId || !NODE_ID_PATTERN.test(nodeId)) {
			error = LOADING_MESSAGES.INVALID_ID;
			loading = false;
			return;
		}

		// Create AbortController for cleanup
		abortController = new AbortController();

		try {
			const response = await fetchWithRetry(
				ROUTES.NODE_CONTENT.replace(':id', nodeId),
				{ signal: abortController.signal }
			);

			if (!response.ok) {
				if (response.status === 404) {
					error = LOADING_MESSAGES.NOT_FOUND;
				} else {
					error = `Failed to load note: ${response.statusText}`;
				}
				loading = false;
				return;
			}

			const data: NodeContentResponse = await response.json();
			nodeTitle = data.title || 'Untitled';
			nodeContent = data.content || '';

			// Parse markdown to HTML and sanitize for security
			const htmlContent = await marked(nodeContent);
			nodeContent = DOMPurify.sanitize(htmlContent, {
				ALLOWED_TAGS: [
					'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
					'p', 'br', 'hr',
					'ul', 'ol', 'li',
					'a', 'img',
					'strong', 'em', 'u', 's',
					'code', 'pre',
					'blockquote',
					'table', 'thead', 'tbody', 'tr', 'th', 'td',
					'div', 'span'
				],
				ALLOWED_ATTR: ['href', 'src', 'alt', 'class', 'id', 'data-note'],
				ALLOW_DATA_ATTR: false
			});

		} catch (err: any) {
			// Don't show error if request was aborted
			if (err?.name !== 'AbortError') {
				error = LOADING_MESSAGES.LOAD_ERROR;
				console.error('Error loading node content:', err);
			}
		} finally {
			loading = false;
			abortController = null;
		}
	});

	// Cleanup on component unmount
	onDestroy(() => {
		if (abortController) {
			abortController.abort();
			abortController = null;
		}
	});

	function handleBackToGraph() {
		goto(ROUTES.GRAPH);
	}
</script>

<svelte:head>
	<title>{nodeTitle || 'Loading...'} - Mnemosyne</title>
</svelte:head>

<div class="note-viewer">
	<header>
		<button class="back-button" on:click={handleBackToGraph}>
			← Back to Graph
		</button>
		{#if !loading && !error}
			<h1>{nodeTitle}</h1>
		{/if}
	</header>

	<main>
		{#if loading}
			<div class="loading">
				<div class="spinner"></div>
				<p>Loading note...</p>
			</div>
		{:else if error}
			<div class="error">
				<p>{error}</p>
				<button on:click={handleBackToGraph}>Return to Graph</button>
			</div>
		{:else}
			<article class="note-content">
				{@html nodeContent}
			</article>
		{/if}
	</main>
</div>

<style>
	.note-viewer {
		height: 100vh;
		display: flex;
		flex-direction: column;
		background-color: var(--color-background);
		color: var(--color-text);
	}

	header {
		background-color: var(--color-surface);
		padding: 1rem 2rem;
		border-bottom: 1px solid rgba(255, 255, 255, 0.1);
		display: flex;
		align-items: center;
		gap: 2rem;
	}

	.back-button {
		background: none;
		border: 1px solid var(--color-primary);
		color: var(--color-primary);
		padding: 0.5rem 1rem;
		border-radius: 4px;
		cursor: pointer;
		font-size: 0.9rem;
		transition: all 0.2s;
	}

	.back-button:hover {
		background-color: var(--color-primary);
		color: white;
	}

	h1 {
		margin: 0;
		font-size: 1.5rem;
		font-weight: 500;
	}

	main {
		flex: 1;
		overflow-y: auto;
		padding: 2rem;
		max-width: 800px;
		width: 100%;
		margin: 0 auto;
	}

	.loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		gap: 1rem;
	}

	.spinner {
		width: 40px;
		height: 40px;
		border: 3px solid rgba(255, 255, 255, 0.1);
		border-top-color: var(--color-primary);
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.error {
		text-align: center;
		padding: 2rem;
	}

	:root {
		--error-color: #e74c3c;
	}

	.error p {
		color: var(--error-color);
		margin-bottom: 1rem;
		font-size: 1.1rem;
	}

	.error button {
		background-color: var(--color-primary);
		color: white;
		border: none;
		padding: 0.75rem 1.5rem;
		border-radius: 4px;
		cursor: pointer;
		font-size: 1rem;
		transition: opacity 0.2s;
	}

	.error button:hover {
		opacity: 0.8;
	}

	.note-content {
		line-height: 1.6;
	}

	/* Markdown content styling */
	.note-content :global(h1),
	.note-content :global(h2),
	.note-content :global(h3),
	.note-content :global(h4),
	.note-content :global(h5),
	.note-content :global(h6) {
		margin-top: 1.5rem;
		margin-bottom: 0.75rem;
		font-weight: 600;
	}

	.note-content :global(p) {
		margin-bottom: 1rem;
	}

	.note-content :global(ul),
	.note-content :global(ol) {
		margin-bottom: 1rem;
		padding-left: 2rem;
	}

	.note-content :global(li) {
		margin-bottom: 0.5rem;
	}

	.note-content :global(code) {
		background-color: var(--color-surface);
		padding: 0.2rem 0.4rem;
		border-radius: 3px;
		font-size: 0.9em;
	}

	.note-content :global(pre) {
		background-color: var(--color-surface);
		padding: 1rem;
		border-radius: 4px;
		overflow-x: auto;
		margin-bottom: 1rem;
	}

	.note-content :global(pre code) {
		background: none;
		padding: 0;
	}

	.note-content :global(blockquote) {
		border-left: 3px solid var(--color-primary);
		padding-left: 1rem;
		margin: 1rem 0;
		color: var(--color-text-secondary);
	}

	.note-content :global(a) {
		color: var(--color-primary);
		text-decoration: none;
		border-bottom: 1px solid transparent;
		transition: border-color 0.2s;
	}

	.note-content :global(a:hover) {
		border-bottom-color: var(--color-primary);
	}

	/* WikiLink specific styling */
	.note-content :global(.wikilink) {
		color: #3498db;
		background-color: rgba(52, 152, 219, 0.1);
		padding: 0.1rem 0.3rem;
		border-radius: 3px;
		border-bottom: none;
		transition: all 0.2s;
	}

	.note-content :global(.wikilink:hover) {
		background-color: rgba(52, 152, 219, 0.2);
		color: #2980b9;
		border-bottom: none;
	}

	.note-content :global(img) {
		max-width: 100%;
		height: auto;
		border-radius: 4px;
		margin: 1rem 0;
	}

	.note-content :global(hr) {
		border: none;
		border-top: 1px solid rgba(255, 255, 255, 0.1);
		margin: 2rem 0;
	}

	.note-content :global(table) {
		width: 100%;
		border-collapse: collapse;
		margin: 1rem 0;
	}

	.note-content :global(th),
	.note-content :global(td) {
		padding: 0.5rem;
		border: 1px solid rgba(255, 255, 255, 0.1);
		text-align: left;
	}

	.note-content :global(th) {
		background-color: var(--color-surface);
		font-weight: 600;
	}
</style>
