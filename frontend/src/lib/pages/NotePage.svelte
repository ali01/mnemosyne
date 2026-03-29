<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { marked } from 'marked';
  import { navigate } from '$lib/router';
  import { wikilinkExtension } from '$lib/utils/wikilink-renderer';
  import DOMPurify from 'isomorphic-dompurify';

  export let graphId: number;
  export let graphUrl: string = '/';
  export let id: string;

  const NODE_ID_PATTERN = /^[a-zA-Z0-9_-]+$/;

  let nodeContent = '';
  let nodeTitle = '';
  let loading = true;
  let error = '';
  let abortController: AbortController | null = null;

  marked.use(wikilinkExtension);

  onMount(async () => {
    if (!id || !NODE_ID_PATTERN.test(id)) {
      error = 'Invalid note ID';
      loading = false;
      return;
    }

    abortController = new AbortController();

    try {
      const response = await fetch(`/api/v1/nodes/${id}/content`, {
        signal: abortController.signal,
      });

      if (!response.ok) {
        error = response.status === 404 ? 'Note not found' : `Failed to load note: ${response.statusText}`;
        loading = false;
        return;
      }

      const data = await response.json();
      nodeTitle = data.title || 'Untitled';

      const htmlContent = await marked(data.content || '');
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
      if (err?.name !== 'AbortError') {
        error = 'Failed to load note content.';
        console.error('Error loading node content:', err);
      }
    } finally {
      loading = false;
      abortController = null;
    }
  });

  onDestroy(() => {
    if (abortController) {
      abortController.abort();
      abortController = null;
    }
  });

  function handleBackToGraph() {
    navigate(graphUrl);
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
    to { transform: rotate(360deg); }
  }

  .error {
    text-align: center;
    padding: 2rem;
  }

  .error p {
    color: #e74c3c;
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
  }

  .note-content { line-height: 1.6; }
  .note-content :global(h1), .note-content :global(h2), .note-content :global(h3) { margin-top: 1.5rem; margin-bottom: 0.75rem; font-weight: 600; }
  .note-content :global(p) { margin-bottom: 1rem; }
  .note-content :global(ul), .note-content :global(ol) { margin-bottom: 1rem; padding-left: 2rem; }
  .note-content :global(code) { background-color: var(--color-surface); padding: 0.2rem 0.4rem; border-radius: 3px; font-size: 0.9em; }
  .note-content :global(pre) { background-color: var(--color-surface); padding: 1rem; border-radius: 4px; overflow-x: auto; margin-bottom: 1rem; }
  .note-content :global(pre code) { background: none; padding: 0; }
  .note-content :global(blockquote) { border-left: 3px solid var(--color-primary); padding-left: 1rem; margin: 1rem 0; color: var(--color-text-secondary); }
  .note-content :global(a) { color: var(--color-primary); text-decoration: none; }
  .note-content :global(.wikilink) { color: #3498db; background-color: rgba(52, 152, 219, 0.1); padding: 0.1rem 0.3rem; border-radius: 3px; }
</style>
