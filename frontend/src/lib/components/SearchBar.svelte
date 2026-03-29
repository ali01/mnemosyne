<script lang="ts">
  import { debounce } from '$lib/utils/debounce';
  import { createEventDispatcher, onDestroy } from 'svelte';

  const MINIMUM_QUERY_LENGTH = 2;
  const SEARCH_DEBOUNCE_MS = 300;
  const BLUR_DELAY_MS = 200;
  const LOADING_SPINNER_DELAY_MS = 100;

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
  let focused = false;

  const performSearch = debounce(async (query: string) => {
    if (!query || query.length < MINIMUM_QUERY_LENGTH) {
      searchResults = [];
      showDropdown = false;
      searchError = '';
      return;
    }

    if (abortController) {
      abortController.abort();
    }
    abortController = new AbortController();

    loading = true;
    searchError = '';

    if (loadingTimeout) clearTimeout(loadingTimeout);
    loadingTimeout = window.setTimeout(() => {
      if (loading) showSpinner = true;
    }, LOADING_SPINNER_DELAY_MS);

    try {
      const response = await fetch(`/api/v1/nodes/search?q=${encodeURIComponent(query)}`, {
        signal: abortController.signal
      });

      if (response.ok) {
        const data = await response.json();
        searchResults = data.nodes || [];
        showDropdown = searchResults.length > 0 || searchError !== '';
        selectedIndex = -1;
      } else {
        searchError = 'Search unavailable';
        searchResults = [];
        showDropdown = true;
      }
    } catch (error) {
      if (error instanceof Error && error.name === 'AbortError') return;
      searchError = 'Connection error';
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

  function handleFocus() {
    focused = true;
  }

  function handleBlur() {
    if (blurTimeout) clearTimeout(blurTimeout);
    blurTimeout = window.setTimeout(() => {
      closeDropdown();
      focused = false;
    }, BLUR_DELAY_MS);
  }

  function getTypeColor(type?: string): string {
    switch (type) {
      case 'core': return 'var(--color-node-core)';
      case 'sub': return 'var(--color-node-sub)';
      case 'detail': return 'var(--color-node-detail)';
      default: return 'var(--color-node-default)';
    }
  }

  onDestroy(() => {
    if (blurTimeout) clearTimeout(blurTimeout);
    if (loadingTimeout) clearTimeout(loadingTimeout);
    if (abortController) abortController.abort();
  });
</script>

<div class="search-container" class:focused>
  <div class="search-input-wrapper">
    <svg class="search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="11" cy="11" r="8"></circle>
      <path d="m21 21-4.35-4.35"></path>
    </svg>
    <input
      bind:this={searchInput}
      bind:value={searchQuery}
      on:input={handleInput}
      on:keydown={handleKeydown}
      on:focus={handleFocus}
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
      <div class="loading-dot"></div>
    {/if}
    {#if !focused && !searchQuery}
      <kbd class="shortcut-hint">/</kbd>
    {/if}
  </div>

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
            <span class="result-dot" style="background: {getTypeColor(result.metadata?.type)}"></span>
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
    width: 100%;
  }

  .search-input-wrapper {
    position: relative;
    display: flex;
    align-items: center;
  }

  .search-icon {
    position: absolute;
    left: 10px;
    color: var(--color-text-muted);
    pointer-events: none;
    transition: color 0.2s var(--ease-out);
  }

  .focused .search-icon {
    color: var(--color-text-secondary);
  }

  .search-input {
    width: 100%;
    padding: 8px 12px 8px 32px;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    color: var(--color-text);
    font-family: var(--font-body);
    font-size: 13px;
    font-weight: 300;
    transition: all 0.25s var(--ease-out);
    letter-spacing: 0.01em;
  }

  .search-input:focus {
    outline: none;
    border-color: var(--color-border-focus);
    background: var(--color-surface-raised);
    box-shadow: 0 0 0 3px var(--color-accent-glow);
  }

  .search-input::placeholder {
    color: var(--color-text-muted);
    font-weight: 300;
  }

  .shortcut-hint {
    position: absolute;
    right: 10px;
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--color-text-muted);
    background: var(--color-surface-raised);
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 1px 6px;
    line-height: 1.4;
    pointer-events: none;
  }

  .loading-dot {
    position: absolute;
    right: 12px;
    width: 6px;
    height: 6px;
    background: var(--color-accent);
    border-radius: 50%;
    animation: pulse 1s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 0.3; transform: scale(0.8); }
    50% { opacity: 1; transform: scale(1.2); }
  }

  .search-dropdown {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    right: 0;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-shadow:
      0 8px 32px rgba(0, 0, 0, 0.4),
      0 0 1px rgba(255, 255, 255, 0.05);
    max-height: 280px;
    overflow-y: auto;
    z-index: 1000;
    backdrop-filter: blur(20px);
  }

  .search-result {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 10px 12px;
    background: none;
    border: none;
    color: var(--color-text);
    text-align: left;
    cursor: pointer;
    transition: background-color 0.15s var(--ease-out);
    font-family: var(--font-body);
  }

  .search-result:hover,
  .search-result.selected {
    background: var(--color-accent-dim);
  }

  .search-result:not(:last-child) {
    border-bottom: 1px solid var(--color-border);
  }

  .result-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .result-title {
    font-size: 13px;
    font-weight: 400;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .result-type {
    font-family: var(--font-mono);
    font-size: 10px;
    color: var(--color-text-muted);
    text-transform: lowercase;
    letter-spacing: 0.03em;
    flex-shrink: 0;
  }

  .search-error {
    padding: 12px;
    color: var(--color-text-secondary);
    font-size: 12px;
    text-align: center;
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
