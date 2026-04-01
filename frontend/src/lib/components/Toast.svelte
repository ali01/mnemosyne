<script lang="ts">
	import { fly } from 'svelte/transition';
	import { toast } from '$lib/stores/toast';

	let toasts: typeof $toast = [];

	$: toasts = $toast;

	function getIcon(type: string) {
		switch (type) {
			case 'success':
				return '✓';
			case 'error':
				return '✕';
			case 'warning':
				return '⚠';
			default:
				return 'ℹ';
		}
	}

	function removeToast(id: string) {
		toast.remove(id);
	}
</script>

<div class="toast-container">
	{#each toasts as toastItem (toastItem.id)}
		<!-- Using assertive aria-live as notifications often contain important/urgent information -->
		<div
			class="toast toast-{toastItem.type}"
			role="alert"
			aria-live="assertive"
			transition:fly={{ y: 50, duration: 300 }}
		>
			<span class="toast-icon">{getIcon(toastItem.type)}</span>
			<!-- Note: Svelte automatically escapes text content, preventing XSS attacks.
			     Only plain text should be passed to toast messages, not HTML content. -->
			<span class="toast-message">{toastItem.message}</span>
			<button
				class="toast-close"
				on:click={() => removeToast(toastItem.id)}
				aria-label="Close notification"
			>
				×
			</button>
		</div>
	{/each}
</div>

<style>
	.toast-container {
		position: fixed;
		bottom: 20px;
		left: 50%;
		transform: translateX(-50%);
		z-index: 1000;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 10px;
		/* Pointer-events strategy: Container has pointer-events: none to allow clicking
		   through the empty space between toasts, while individual toasts have
		   pointer-events: all to remain interactive. This prevents the container from
		   blocking interactions with underlying UI elements. */
		pointer-events: none;
	}

	.toast {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px 16px;
		background-color: var(--color-surface);
		border-radius: 8px;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		color: white;
		font-size: 0.9rem;
		min-width: 300px;
		max-width: 500px;
		pointer-events: all;
		border-left: 4px solid;
	}

	.toast-success {
		border-left-color: #2ecc71;
	}

	.toast-error {
		border-left-color: #e74c3c;
	}

	.toast-warning {
		border-left-color: #f39c12;
	}

	.toast-info {
		border-left-color: #3498db;
	}

	.toast-icon {
		font-size: 1.2rem;
		width: 24px;
		height: 24px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.toast-success .toast-icon {
		background-color: #2ecc71;
	}

	.toast-error .toast-icon {
		background-color: #e74c3c;
	}

	.toast-warning .toast-icon {
		background-color: #f39c12;
	}

	.toast-info .toast-icon {
		background-color: #3498db;
	}

	.toast-message {
		flex: 1;
		word-wrap: break-word;
	}

	.toast-close {
		background: none;
		border: none;
		color: var(--color-text-secondary);
		font-size: 1.5rem;
		cursor: pointer;
		padding: 0;
		width: 24px;
		height: 24px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 4px;
		transition: all 0.2s;
		flex-shrink: 0;
	}

	.toast-close:hover {
		background-color: rgba(255, 255, 255, 0.1);
		color: white;
	}
</style>
