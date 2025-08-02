import { writable } from 'svelte/store';

// Constants
const DEFAULT_TOAST_DURATION = 5000;

export interface Toast {
  id: string;
  message: string;
  type: 'success' | 'error' | 'info' | 'warning';
  duration?: number;
}

function createToastStore() {
  const { subscribe, update } = writable<Toast[]>([]);
  const timeouts = new Map<string, number>();

  let toastId = 0;

  function addToast(message: string, type: Toast['type'] = 'info', duration = DEFAULT_TOAST_DURATION) {
    const id = `toast-${toastId++}`;
    const toast: Toast = { id, message, type, duration };

    update(toasts => [...toasts, toast]);

    // Set auto-removal timeout and track it
    if (duration > 0) {
      const timeoutId = window.setTimeout(() => {
        removeToast(id);
      }, duration);
      timeouts.set(id, timeoutId);
    }

    return id;
  }

  function removeToast(id: string) {
    // Clear any pending timeout for this toast
    const timeoutId = timeouts.get(id);
    if (timeoutId !== undefined) {
      clearTimeout(timeoutId);
      timeouts.delete(id);
    }

    update(toasts => toasts.filter(t => t.id !== id));
  }

  return {
    subscribe,
    success: (message: string, duration?: number) => addToast(message, 'success', duration),
    error: (message: string, duration?: number) => addToast(message, 'error', duration),
    info: (message: string, duration?: number) => addToast(message, 'info', duration),
    warning: (message: string, duration?: number) => addToast(message, 'warning', duration),
    remove: removeToast
  };
}

export const toast = createToastStore();
