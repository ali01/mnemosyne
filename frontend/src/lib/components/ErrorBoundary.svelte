<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from '$lib/stores/toast';
	
	export let fallback = 'Something went wrong. Please refresh the page.';
	
	let hasError = false;
	let errorMessage = '';
	
	onMount(() => {
		// Global error handler
		const handleError = (event: ErrorEvent) => {
			console.error('Uncaught error:', event.error);
			hasError = true;
			errorMessage = event.error?.message || fallback;
			toast.error('An unexpected error occurred');
			event.preventDefault();
		};
		
		// Promise rejection handler
		const handleRejection = (event: PromiseRejectionEvent) => {
			console.error('Unhandled promise rejection:', event.reason);
			hasError = true;
			errorMessage = event.reason?.message || fallback;
			toast.error('An unexpected error occurred');
			event.preventDefault();
		};
		
		window.addEventListener('error', handleError);
		window.addEventListener('unhandledrejection', handleRejection);
		
		return () => {
			window.removeEventListener('error', handleError);
			window.removeEventListener('unhandledrejection', handleRejection);
		};
	});
	
	function handleReload() {
		window.location.reload();
	}
</script>

{#if hasError}
	<div class="error-boundary">
		<div class="error-content">
			<h2>Oops! Something went wrong</h2>
			<p>{errorMessage}</p>
			<button on:click={handleReload}>Reload Page</button>
		</div>
	</div>
{:else}
	<slot />
{/if}

<style>
	.error-boundary {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background-color: var(--color-background);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 9999;
	}
	
	.error-content {
		text-align: center;
		padding: 2rem;
		max-width: 500px;
	}
	
	h2 {
		color: #e74c3c;
		margin-bottom: 1rem;
		font-size: 1.5rem;
	}
	
	p {
		color: var(--color-text-secondary);
		margin-bottom: 2rem;
		line-height: 1.6;
	}
	
	button {
		background-color: var(--color-primary);
		color: white;
		border: none;
		padding: 0.75rem 2rem;
		border-radius: 4px;
		cursor: pointer;
		font-size: 1rem;
		transition: opacity 0.2s;
	}
	
	button:hover {
		opacity: 0.8;
	}
</style>