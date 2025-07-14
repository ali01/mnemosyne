<script lang="ts">
	export let size: 'small' | 'medium' | 'large' = 'medium';
	export let message = '';
</script>

<div class="loading-spinner loading-spinner-{size}" role="status" aria-live="polite">
	<div class="spinner" aria-hidden="true"></div>
	{#if message}
		<p class="loading-message">{message}</p>
	{/if}
	<span class="sr-only">Loading...</span>
</div>

<style>
	.loading-spinner {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 1rem;
	}

	.spinner {
		border: 3px solid rgba(255, 255, 255, 0.1);
		border-top-color: var(--color-primary);
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	.loading-spinner-small .spinner {
		width: 24px;
		height: 24px;
		border-width: 2px;
	}

	.loading-spinner-medium .spinner {
		width: 40px;
		height: 40px;
	}

	.loading-spinner-large .spinner {
		width: 60px;
		height: 60px;
		border-width: 4px;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.loading-message {
		color: var(--color-text-secondary);
		font-size: 0.9rem;
		margin: 0;
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
		border: 0;
	}
</style>
