<script lang="ts">
  export let size: 'small' | 'medium' | 'large' = 'medium';
  export let message = '';
</script>

<div class="loading-spinner loading-spinner-{size}" role="status" aria-live="polite">
  <div class="spinner" aria-hidden="true">
    <div class="dot dot-1"></div>
    <div class="dot dot-2"></div>
    <div class="dot dot-3"></div>
  </div>
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
    gap: 16px;
  }

  .spinner {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .dot {
    border-radius: 50%;
    background: var(--color-accent, #7b8cff);
    animation: dotPulse 1.4s ease-in-out infinite;
  }

  .dot-2 { animation-delay: 0.15s; }
  .dot-3 { animation-delay: 0.3s; }

  .loading-spinner-small .dot {
    width: 4px;
    height: 4px;
  }

  .loading-spinner-medium .dot {
    width: 6px;
    height: 6px;
  }

  .loading-spinner-large .dot {
    width: 8px;
    height: 8px;
  }

  @keyframes dotPulse {
    0%, 80%, 100% {
      opacity: 0.2;
      transform: scale(0.8);
    }
    40% {
      opacity: 1;
      transform: scale(1);
    }
  }

  .loading-message {
    color: var(--color-text-muted, #444650);
    font-family: var(--font-body, 'DM Sans', sans-serif);
    font-size: 12px;
    font-weight: 300;
    margin: 0;
    text-align: center;
    letter-spacing: 0.03em;
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
